package tasks

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/manager"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/pkg"
	"commander-server-t260220/internal/types"
	"commander-server-t260220/internal/types/toc"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/datatypes"
)

func listing1688Enabled(payload *toc.ProductIssue) bool {
	if payload.EnableImg2Img == nil {
		return true
	}
	return *payload.EnableImg2Img
}

func marshalJSONStrings(list []string) datatypes.JSON {
	b, _ := json.Marshal(list)
	return datatypes.JSON(b)
}

func (t *Task) failListing1688Job(jobID uint, code constants.Listing1688ErrorCode, msg string) {
	if t.listing1688JobRepo == nil || jobID == 0 {
		return
	}
	_ = t.listing1688JobRepo.Update(jobID, map[string]interface{}{
		"status":     constants.StatusFailed,
		"error_code": string(code),
		"message":    msg,
	})
}

func (t *Task) runListing1688Img2Img(sourceURLs []string) ([]toc.ProductIssueImage, error) {
	images := make([]toc.ProductIssueImage, len(sourceURLs))
	// 上游 hyhacct（与 POST /api/v1/ai/produce_images 同源）；禁止 1688 走 Hyhacct / Temu 轮播编排。
	model := ""
	if t.config != nil {
		model = strings.TrimSpace(t.config.Hyhacct.ModelImages)
	}
	for i, src := range sourceURLs {
		images[i] = toc.ProductIssueImage{
			Prompt: constants.Listing1688Img2ImgPrompt,
			Status: constants.StatusProcessing,
		}
		refBytes, err := pkg.DecodeImageInputToBytes(src)
		if err != nil {
			images[i].Status = constants.StatusFailed
			images[i].Message = "解析参考图失败: " + err.Error()
			continue
		}
		url, err := utils.AiEditImage(t.hyhacctUtils, constants.Listing1688Img2ImgPrompt, [][]byte{refBytes}, "1024:1024", "1K", model)
		if err != nil {
			images[i].Status = constants.StatusFailed
			images[i].Message = "生成失败: " + err.Error()
			continue
		}
		images[i].Status = constants.StatusSuccess
		images[i].Url = url
		images[i].Message = "完成"
	}
	successCnt, failedCnt := 0, 0
	for _, im := range images {
		if im.Status == constants.StatusSuccess && strings.TrimSpace(im.Url) != "" {
			successCnt++
		} else {
			failedCnt++
		}
	}
	if successCnt == 0 {
		return images, fmt.Errorf("%s", constants.Listing1688ErrImg2ImgFailed)
	}
	if failedCnt > 0 {
		return images, fmt.Errorf("%s", constants.Listing1688ErrImg2ImgPartialFail)
	}
	return images, nil
}

// RunListing1688 1688 采集图优旁路：Onebound 拉图 → hyhacct 图生图（produce_images 同源）→ 下发 Agent（不走 Temu 五场景 LLM）。
func (t *Task) RunListing1688(ctx manager.TaskContext) error {
	var payload toc.ProductIssue
	if err := json.Unmarshal(ctx.SendJSON, &payload); err != nil {
		return fmt.Errorf("解析 Send 失败: %w", err)
	}
	peerURL := strings.TrimSpace(payload.PeerUrl)
	if peerURL == "" {
		return fmt.Errorf("%s", constants.Listing1688ErrInvalidPeerURL)
	}

	var jobID uint
	if t.listing1688JobRepo != nil {
		row := &modules.TableListing1688Jobs{
			UserID:  ctx.UserId,
			TaskID:  ctx.TaskId,
			PeerURL: peerURL,
			Status:  constants.StatusProcessing,
			Message: "采集中...",
		}
		if err := t.listing1688JobRepo.Create(row); err != nil {
			t.logger.Error("创建 1688 审计记录失败: %s", err.Error())
		} else {
			jobID = row.ID
		}
	}

	ctx.Progress.UpdateMessage(ctx.TaskId, "正在采集同行商品图片...")
	numIid, title, sourceURLs, err := CollectPeerImagesFromURL(t.oneboundUtils, peerURL)
	if err != nil {
		code := constants.Listing1688ErrCollectFailed
		if IsListing1688CollectNoImageErr(err) {
			code = constants.Listing1688ErrCollectNoImage
		} else if _, parseErr := Parse1688NumIid(peerURL); parseErr != nil {
			code = constants.Listing1688ErrInvalidPeerURL
		}
		t.failListing1688Job(jobID, code, err.Error())
		return fmt.Errorf("%s: %w", code, err)
	}

	payload.SourceImages = sourceURLs

	if t.listing1688JobRepo != nil && jobID != 0 {
		_ = t.listing1688JobRepo.Update(jobID, map[string]interface{}{
			"num_iid":        numIid,
			"source_images":  marshalJSONStrings(sourceURLs),
			"message":        "采集完成，准备图生图...",
		})
	}

	if listing1688Enabled(&payload) {
		ctx.Progress.UpdateMessage(ctx.TaskId, fmt.Sprintf("正在图生图优化 %d 张商品图...", len(sourceURLs)))
		images, imgErr := t.runListing1688Img2Img(sourceURLs)
		payload.Images = images
		if imgErr != nil {
			code := constants.Listing1688ErrImg2ImgFailed
			if strings.Contains(imgErr.Error(), string(constants.Listing1688ErrImg2ImgPartialFail)) {
				code = constants.Listing1688ErrImg2ImgPartialFail
			}
			t.failListing1688Job(jobID, code, imgErr.Error())
			return imgErr
		}
	} else {
		images := make([]toc.ProductIssueImage, len(sourceURLs))
		for i, u := range sourceURLs {
			images[i] = toc.ProductIssueImage{Url: u, Status: constants.StatusSuccess, Message: "沿用原图"}
		}
		payload.Images = images
	}

	if strings.TrimSpace(payload.TitleZh) == "" && title != "" {
		payload.TitleZh = title
	}
	if len(payload.Variants) == 0 {
		payload.Variants = []toc.ProductIssueVariant{{Model: "默认", Count: "1", Price: 100}}
	}
	fillVariantURLFromFirstCarousel(&payload)

	optURLs := make([]string, 0, len(payload.Images))
	for _, im := range payload.Images {
		if im.Status == constants.StatusSuccess && strings.TrimSpace(im.Url) != "" {
			optURLs = append(optURLs, im.Url)
		}
	}
	if t.listing1688JobRepo != nil && jobID != 0 {
		_ = t.listing1688JobRepo.Update(jobID, map[string]interface{}{
			"optimized_images": marshalJSONStrings(optURLs),
			"message":          "图优完成，提交 Agent...",
		})
	}

	ctx.Progress.UpdateSendOrReceive(ctx.TaskId, payload, nil)

	agent := t.agentManager.GetAgent(ctx.AgentId)
	if agent == nil {
		t.failListing1688Job(jobID, constants.Listing1688ErrAgentOffline, "Agent 不存在或已断开连接")
		return fmt.Errorf("%s", constants.Listing1688ErrAgentOffline)
	}

	agentPayload := payload
	agentPayload.ImageRawUrl = ""
	data := types.AgentMessage[any, any]{
		TaskId:   ctx.TaskId,
		RunAt:    ctx.RunAt,
		AgentId:  ctx.AgentId,
		Platform: ctx.Platform,
		Protocol: ctx.Protocol,
		Status:   constants.StatusProcessing,
		Message:  "已提交到 Agent",
		Send:     agentPayload,
		Receive:  nil,
	}
	if err := t.agentManager.AsyncSend(ctx.AgentId, data); err != nil {
		t.failListing1688Job(jobID, constants.Listing1688ErrInternal, err.Error())
		t.logger.Error("发送任务到 Agent 失败: %s", err.Error())
		return err
	}
	ctx.Progress.UpdateMessage(ctx.TaskId, "已提交 Agent，正在妙手 1688 上架...")
	return nil
}
