package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type listing1688PublishReq struct {
	Title string `json:"title"`
}

// Listing1688Publish 经 WorkerManager 创建 system_tasks 并调度商家发品（禁止妙手/ProductIssue）
func (s *Service) Listing1688Publish(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if err := s.listing1688AssertImgReady(row); err != nil {
		s.response.Forbidden(ctx, err.Error(), err)
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	if _, err := s.listing1688RequireSellerSession(row); err != nil {
		code := constants.ParseListing1688WizardErrorCode(err.Error())
		if code == "" {
			if strings.Contains(err.Error(), string(constants.Listing1688ErrAgentOffline)) {
				code = constants.Listing1688ErrWizardAgentOffline
			} else {
				code = constants.Listing1688ErrSellerSessionInvalid
			}
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(code, ""), err)
		return
	}

	draft, err := s.listing1688PublishDraftRepo().GetByWorkflow(row.ID)
	if err != nil || !listing1688PublishDraftCanSubmit(draft.ValidationStatus, draft.DraftSaved) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "请先完成类目表单校验并保存草稿"), err)
		return
	}
	formData := listing1688DecodePublishFieldsFromJSON(draft.FormData)
	title, _ := formData["title"].(string)
	title = strings.TrimSpace(title)
	images, _ := s.listing1688SelectedImagesAndTitle(row)
	if title == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "发布草案标题为空"), nil)
		return
	}
	if len(images) < 1 {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, ""), nil)
		return
	}

	sendPayload := map[string]any{
		"workflow_id":    row.ID,
		"title":          title,
		"images":         images,
		"offer_url":      row.OfferURL,
		"skus":           listing1688SnapshotSkus(row.OfferSnapshot),
		"publish_form":   formData,
		"category_id":    draft.CategoryID,
		"draft_verified": true,
	}
	if s.workerManager == nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": WorkerManager 未初始化", nil)
		return
	}
	taskID, err := s.workerManager.CreateNewTaskID(row.AgentID, constants.ProtocolListing1688SellerPublish, constants.Platform1688, sendPayload, row.UserID)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 创建发品任务失败", err)
		return
	}
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":          string(constants.Listing1688StatusPublishing),
		"publish_task_id": taskID,
		"error_code":      "",
		"message":         "商家发品进行中",
	})

	s.response.Success(ctx, gin.H{
		"workflow_id":     row.ID,
		"status":          constants.Listing1688StatusPublishing,
		"publish_task_id": taskID,
		"message":         "已入队商家发品",
	})
}

// Listing1688PublishStatus 查询发品任务与 workflow 状态
func (s *Service) Listing1688PublishStatus(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	out := gin.H{
		"workflow_id":     row.ID,
		"status":          row.Status,
		"publish_task_id": row.PublishTaskID,
		"error_code":      row.ErrorCode,
		"message":         row.Message,
	}
	if row.PublishTaskID != "" && s.systemTaskRepo != nil {
		if t, err := s.systemTaskRepo.GetByTaskId(row.PublishTaskID); err == nil && t != nil {
			out["task_status"] = t.Status
			out["task_message"] = t.Message
		}
	}
	s.response.Success(ctx, out)
}

// listing1688SelectedImagesAndTitle 以 selected jobs / selected_image_idx 为准，禁止盲信 snapshot 图数
func (s *Service) listing1688SelectedImagesAndTitle(row *modules.TableListing1688Workflows) ([]string, string) {
	images := make([]string, 0)
	title := ""
	if row == nil {
		return images, title
	}
	var snap map[string]any
	if len(row.OfferSnapshot) > 0 {
		_ = json.Unmarshal(row.OfferSnapshot, &snap)
	}
	if snap != nil {
		if t, ok := snap["title"].(string); ok {
			title = strings.TrimSpace(t)
		}
	}
	jobs, err := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	if err == nil {
		for _, j := range jobs {
			if j.Status == constants.Listing1688ImageSelected && strings.TrimSpace(j.ResultURL) != "" {
				images = append(images, strings.TrimSpace(j.ResultURL))
			}
		}
	}
	if len(images) == 0 && len(row.SelectedImageIdx) > 0 {
		var sel []listing1688SelectedImage
		if json.Unmarshal(row.SelectedImageIdx, &sel) == nil {
			for _, item := range sel {
				u := strings.TrimSpace(item.ResultURL)
				if u != "" {
					images = append(images, u)
				}
			}
		}
	}
	return images, title
}

func listing1688SnapshotSkus(raw datatypes.JSON) []map[string]any {
	out := make([]map[string]any, 0)
	if len(raw) == 0 {
		return out
	}
	var snap map[string]any
	if err := json.Unmarshal(raw, &snap); err != nil || snap == nil {
		return out
	}
	arr, ok := snap["skus"].([]any)
	if !ok {
		return out
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, m)
	}
	return out
}
