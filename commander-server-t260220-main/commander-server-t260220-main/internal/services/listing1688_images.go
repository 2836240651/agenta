package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/pkg"
	"commander-server-t260220/internal/repos"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type listing1688ImagesReq struct {
	Slot   int    `json:"slot"`
	Prompt string `json:"prompt"`
}

type listing1688SelectedImage struct {
	Slot      int    `json:"slot"`
	ResultURL string `json:"result_url"`
}

type listing1688ImageJobStore interface {
	Create(row *modules.TableListing1688ImageJobs) error
	Update(id uint, updates map[string]interface{}) error
	ListByWorkflow(workflowID uint) ([]modules.TableListing1688ImageJobs, error)
	GetByWorkflowSlot(workflowID uint, slot int) (*modules.TableListing1688ImageJobs, error)
}

func (s *Service) listing1688ImageJobs() listing1688ImageJobStore {
	if s.testListing1688ImageJobs != nil {
		return s.testListing1688ImageJobs
	}
	return repos.NewListing1688ImageJobRepo(s.gormUtils.Get())
}

func (s *Service) listing1688DecodeImage(src string) ([]byte, error) {
	if s.testListing1688DecodeImage != nil {
		return s.testListing1688DecodeImage(src)
	}
	return pkg.DecodeImageInputToBytes(src)
}

func (s *Service) listing1688RunAiEdit(prompt string, ref [][]byte) (string, error) {
	if s.testListing1688ImageEdit != nil {
		return s.testListing1688ImageEdit(prompt, ref)
	}
	model := ""
	if s.config != nil {
		model = strings.TrimSpace(s.config.ServerConfig.Hyhacct.ModelImages)
	}
	return utils.AiEditImage(s.hyhacctUtils, prompt, ref, "1024:1024", "1K", model)
}

func listing1688TruncateSourceURLs(urls []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, constants.Listing1688MaxImageSlots)
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
		if len(out) >= constants.Listing1688MaxImageSlots {
			break
		}
	}
	return out
}

func listing1688ParseSnapshotMainImages(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("offer_snapshot 为空")
	}
	var snap struct {
		MainImages []string `json:"main_images"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		return nil, err
	}
	urls := listing1688TruncateSourceURLs(snap.MainImages)
	if len(urls) < 1 {
		return nil, fmt.Errorf("无可用主图")
	}
	return urls, nil
}

func listing1688JobPhaseOK(status string) bool {
	switch status {
	case string(constants.Listing1688StatusCollected),
		string(constants.Listing1688StatusImgEditing),
		string(constants.Listing1688StatusImgReady):
		return true
	default:
		return false
	}
}

func listing1688JobToMap(j *modules.TableListing1688ImageJobs) gin.H {
	if j == nil {
		return nil
	}
	return gin.H{
		"id":         j.ID,
		"slot":       j.Slot,
		"source_url": j.SourceURL,
		"result_url": j.ResultURL,
		"prompt":     j.Prompt,
		"status":     j.Status,
		"error_code": j.ErrorCode,
		"message":    j.Message,
	}
}

func listing1688JobsToMaps(jobs []modules.TableListing1688ImageJobs) []gin.H {
	out := make([]gin.H, 0, len(jobs))
	for i := range jobs {
		out = append(out, listing1688JobToMap(&jobs[i]))
	}
	return out
}

// listing1688EnsureImageSlots 从 offer_snapshot 懒创建 Slot（最多 10）
func (s *Service) listing1688EnsureImageSlots(row *modules.TableListing1688Workflows) ([]modules.TableListing1688ImageJobs, error) {
	existing, err := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing, nil
	}
	urls, err := listing1688ParseSnapshotMainImages(row.OfferSnapshot)
	if err != nil {
		return nil, err
	}
	for i, src := range urls {
		job := &modules.TableListing1688ImageJobs{
			WorkflowID: row.ID,
			UserID:     row.UserID,
			Slot:       i,
			SourceURL:  src,
			Prompt:     constants.Listing1688WizardDefaultPrompt,
			Status:     constants.Listing1688ImagePending,
		}
		if err := s.listing1688ImageJobs().Create(job); err != nil {
			return nil, err
		}
	}
	return s.listing1688ImageJobs().ListByWorkflow(row.ID)
}

func (s *Service) listing1688ImagesResponse(ctx *gin.Context, row *modules.TableListing1688Workflows, slot int, job *modules.TableListing1688ImageJobs) {
	jobs, _ := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	fresh, _ := s.listing1688Workflows().GetByIDForUser(row.ID, row.UserID)
	status := row.Status
	sel := row.SelectedImageIdx
	if fresh != nil {
		status = fresh.Status
		sel = fresh.SelectedImageIdx
	}
	s.response.Success(ctx, gin.H{
		"workflow_id":        row.ID,
		"status":             status,
		"slot":               slot,
		"job":                listing1688JobToMap(job),
		"image_jobs":         listing1688JobsToMaps(jobs),
		"selected_image_idx": sel,
	})
}

// listing1688RecomputeImageReady 全 Slot selected → img_ready，否则保持/回落 img_editing
func (s *Service) listing1688RecomputeImageReady(row *modules.TableListing1688Workflows) (string, datatypes.JSON, error) {
	jobs, err := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	if err != nil {
		return "", nil, err
	}
	selected := make([]listing1688SelectedImage, 0)
	allSelected := len(jobs) > 0
	for _, j := range jobs {
		if j.Status != constants.Listing1688ImageSelected {
			allSelected = false
			continue
		}
		selected = append(selected, listing1688SelectedImage{Slot: j.Slot, ResultURL: j.ResultURL})
	}
	raw, _ := json.Marshal(selected)
	status := string(constants.Listing1688StatusImgEditing)
	if allSelected {
		status = string(constants.Listing1688StatusImgReady)
	}
	if err := s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":             status,
		"selected_image_idx": datatypes.JSON(raw),
		"error_code":         "",
		"message":            "",
	}); err != nil {
		return "", nil, err
	}
	return status, datatypes.JSON(raw), nil
}

// listing1688AssertImgReady M3 发品门禁；未就绪返回 IMAGES_NOT_READY（以 jobs 为准，不盲信 status）
func (s *Service) listing1688AssertImgReady(row *modules.TableListing1688Workflows) error {
	if row == nil {
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, "workflow 为空"))
	}
	jobs, err := s.listing1688ImageJobs().ListByWorkflow(row.ID)
	if err != nil {
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, err.Error()))
	}
	if len(jobs) == 0 {
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, "尚未图优"))
	}
	for _, j := range jobs {
		if j.Status != constants.Listing1688ImageSelected {
			return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, fmt.Sprintf("slot %d 未选定", j.Slot)))
		}
	}
	return nil
}

func (s *Service) listing1688DoImageGenerate(ctx *gin.Context, isRegen bool) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !listing1688JobPhaseOK(row.Status) {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": 请先完成竞品采集", nil)
		return
	}
	var req listing1688ImagesReq
	_ = ctx.ShouldBindJSON(&req)
	if req.Slot < 0 || req.Slot >= constants.Listing1688MaxImageSlots {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+fmt.Sprintf(": slot 超出上限（0-%d）", constants.Listing1688MaxImageSlots-1), nil)
		return
	}

	if _, err := s.listing1688EnsureImageSlots(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": "+err.Error(), err)
		return
	}
	job, err := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": slot 不存在（已超过采集图数量）", err)
			return
		}
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 查询 slot 失败", err)
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(job.Prompt)
	}
	if prompt == "" {
		prompt = constants.Listing1688WizardDefaultPrompt
	}

	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status": string(constants.Listing1688StatusImgEditing),
	})
	_ = s.listing1688ImageJobs().Update(job.ID, map[string]interface{}{
		"status":     constants.Listing1688ImageRunning,
		"prompt":     prompt,
		"error_code": "",
		"message":    "处理中",
		"result_url": "",
	})
	job.Status = constants.Listing1688ImageRunning
	job.Prompt = prompt
	job.Message = "处理中"
	job.ResultURL = ""
	_ = isRegen // regenerate 与 generate 同路径，仅语义区分

	// 异步执行上游图生图：避免 HTTP 同步等待 30–180s+ 触发前端 axios 300s 超时。
	wfID := row.ID
	jobID := job.ID
	sourceURL := job.SourceURL
	promptCopy := prompt
	go func() {
		wf := &modules.TableListing1688Workflows{Model: gorm.Model{ID: wfID}}
		ref, err := s.listing1688DecodeImage(sourceURL)
		if err != nil {
			// IMPORTANT:
			// Keep job status as Running until selected_image_idx is recomputed.
			// This avoids a race in tests where the failure is observed (status != Running)
			// before the recompute clears selected_image_idx.
			_ = s.listing1688ImageJobs().Update(jobID, map[string]interface{}{
				"error_code": string(constants.Listing1688ErrImg2ImgFailed),
				"message":    "参考图下载失败: " + err.Error(),
			})
			_, _, _ = s.listing1688RecomputeImageReady(wf)
			_ = s.listing1688ImageJobs().Update(jobID, map[string]interface{}{
				"status":     constants.Listing1688ImageFailed,
				"error_code": string(constants.Listing1688ErrImg2ImgFailed),
				"message":    "参考图下载失败: " + err.Error(),
			})
			return
		}
		resultURL, err := s.listing1688RunAiEdit(promptCopy, [][]byte{ref})
		if err != nil {
			_ = s.listing1688ImageJobs().Update(jobID, map[string]interface{}{
				"error_code": string(constants.Listing1688ErrImg2ImgFailed),
				"message":    err.Error(),
			})
			_, _, _ = s.listing1688RecomputeImageReady(wf)
			_ = s.listing1688ImageJobs().Update(jobID, map[string]interface{}{
				"status":     constants.Listing1688ImageFailed,
				"error_code": string(constants.Listing1688ErrImg2ImgFailed),
				"message":    err.Error(),
			})
			return
		}
		_ = s.listing1688ImageJobs().Update(jobID, map[string]interface{}{
			"status":     constants.Listing1688ImageDone,
			"result_url": resultURL,
			"prompt":     promptCopy,
			"error_code": "",
			"message":    "",
		})
		_, _, _ = s.listing1688RecomputeImageReady(wf)
	}()

	freshJob, _ := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if freshJob != nil {
		job = freshJob
	}
	s.listing1688ImagesResponse(ctx, row, req.Slot, job)
}

func (s *Service) Listing1688ImagesGenerate(ctx *gin.Context) {
	s.listing1688DoImageGenerate(ctx, false)
}

func (s *Service) Listing1688ImagesRegenerate(ctx *gin.Context) {
	s.listing1688DoImageGenerate(ctx, true)
}

func (s *Service) Listing1688ImagesUseOriginal(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !listing1688JobPhaseOK(row.Status) {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": 请先完成竞品采集", nil)
		return
	}
	var req listing1688ImagesReq
	_ = ctx.ShouldBindJSON(&req)
	if req.Slot < 0 || req.Slot >= constants.Listing1688MaxImageSlots {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": slot 超出上限", nil)
		return
	}
	if _, err := s.listing1688EnsureImageSlots(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": "+err.Error(), err)
		return
	}
	job, err := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": slot 不存在", err)
		return
	}
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status": string(constants.Listing1688StatusImgEditing),
	})
	_ = s.listing1688ImageJobs().Update(job.ID, map[string]interface{}{
		"status":     constants.Listing1688ImageUseOriginal,
		"result_url": job.SourceURL,
		"error_code": "",
		"message":    "",
	})
	job.Status = constants.Listing1688ImageUseOriginal
	job.ResultURL = job.SourceURL
	_, _, _ = s.listing1688RecomputeImageReady(row)
	freshJob, _ := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if freshJob != nil {
		job = freshJob
	}
	s.listing1688ImagesResponse(ctx, row, req.Slot, job)
}

func (s *Service) Listing1688ImagesSelect(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !listing1688JobPhaseOK(row.Status) {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": 请先完成竞品采集", nil)
		return
	}
	var req listing1688ImagesReq
	_ = ctx.ShouldBindJSON(&req)
	if req.Slot < 0 || req.Slot >= constants.Listing1688MaxImageSlots {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": slot 超出上限", nil)
		return
	}
	if _, err := s.listing1688EnsureImageSlots(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrCollectParseFail)+": "+err.Error(), err)
		return
	}
	job, err := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": slot 不存在", err)
		return
	}
	if job.Status != constants.Listing1688ImageDone && job.Status != constants.Listing1688ImageUseOriginal && job.Status != constants.Listing1688ImageSelected {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrImagesNotReady)+": 请先生成或沿用原图", nil)
		return
	}
	if strings.TrimSpace(job.ResultURL) == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrImagesNotReady)+": 结果图为空", nil)
		return
	}
	_ = s.listing1688ImageJobs().Update(job.ID, map[string]interface{}{
		"status": constants.Listing1688ImageSelected,
	})
	job.Status = constants.Listing1688ImageSelected
	_, _, _ = s.listing1688RecomputeImageReady(row)
	freshJob, _ := s.listing1688ImageJobs().GetByWorkflowSlot(row.ID, req.Slot)
	if freshJob != nil {
		job = freshJob
	}
	s.listing1688ImagesResponse(ctx, row, req.Slot, job)
}
