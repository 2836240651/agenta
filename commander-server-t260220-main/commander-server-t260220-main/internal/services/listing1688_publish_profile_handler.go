package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/repos"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type listing1688StorePresetReq struct {
	StoreKey string         `json:"store_key"`
	Fields   map[string]any `json:"fields"`
	IsActive *bool          `json:"is_active"`
}
type listing1688CategoryTemplateReq struct {
	StoreKey       string         `json:"store_key"`
	CategoryID     string         `json:"category_id"`
	CategoryPath   string         `json:"category_path"`
	ProductFamily  string         `json:"product_family"`
	Fields         map[string]any `json:"fields"`
	RequiredFields []string       `json:"required_fields"`
	SkuRules       map[string]any `json:"sku_rules"`
	DetailTemplate map[string]any `json:"detail_template"`
	IsActive       *bool          `json:"is_active"`
}
type listing1688PublishFormValidateReq struct {
	StoreKey   string         `json:"store_key"`
	CategoryID string         `json:"category_id"`
	Fields     map[string]any `json:"fields"`
}

func (s *Service) listing1688StorePresetRepo() *repos.Listing1688StorePresetRepo {
	return repos.NewListing1688StorePresetRepo(s.gormUtils.Get())
}
func (s *Service) listing1688CategoryTemplateRepo() *repos.Listing1688CategoryTemplateRepo {
	return repos.NewListing1688CategoryTemplateRepo(s.gormUtils.Get())
}
func (s *Service) listing1688PublishDraftRepo() *repos.Listing1688PublishDraftRepo {
	return repos.NewListing1688PublishDraftRepo(s.gormUtils.Get())
}
func listing1688StoreKey(raw string) string {
	if key := strings.TrimSpace(raw); key != "" {
		return key
	}
	return "default"
}

func (s *Service) Listing1688StorePresetGet(ctx *gin.Context) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return
	}
	storeKey := listing1688StoreKey(ctx.Query("store_key"))
	row, err := s.listing1688StorePresetRepo().GetByUserStore(userID, storeKey)
	if err != nil {
		s.response.Success(ctx, gin.H{"store_key": storeKey, "fields": map[string]any{}, "is_active": true})
		return
	}
	s.response.Success(ctx, row)
}

func (s *Service) Listing1688StorePresetUpsert(ctx *gin.Context) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return
	}
	var req listing1688StorePresetReq
	_ = ctx.ShouldBindJSON(&req)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	fields, _ := json.Marshal(req.Fields)
	row := &modules.TableListing1688StorePresets{UserID: userID, StoreKey: listing1688StoreKey(req.StoreKey), Fields: datatypes.JSON(fields), IsActive: isActive}
	if err := s.listing1688StorePresetRepo().Upsert(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": save store preset failed", err)
		return
	}
	s.response.Success(ctx, row)
}

func (s *Service) Listing1688CategoryTemplateList(ctx *gin.Context) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return
	}
	rows, err := s.listing1688CategoryTemplateRepo().ListByUserStore(userID, listing1688StoreKey(ctx.Query("store_key")))
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": load category templates failed", err)
		return
	}
	s.response.Success(ctx, rows)
}

func (s *Service) Listing1688CategoryTemplateUpsert(ctx *gin.Context) {
	userID, ok := s.listing1688RequireUser(ctx)
	if !ok {
		return
	}
	var req listing1688CategoryTemplateReq
	_ = ctx.ShouldBindJSON(&req)
	categoryID := strings.TrimSpace(req.CategoryID)
	if categoryID == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": category_id is required", nil)
		return
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	fields, _ := json.Marshal(req.Fields)
	requiredFields, _ := json.Marshal(req.RequiredFields)
	skuRules, _ := json.Marshal(req.SkuRules)
	detailTemplate, _ := json.Marshal(req.DetailTemplate)
	row := &modules.TableListing1688CategoryTemplates{UserID: userID, StoreKey: listing1688StoreKey(req.StoreKey), CategoryID: categoryID, CategoryPath: strings.TrimSpace(req.CategoryPath), ProductFamily: strings.TrimSpace(req.ProductFamily), Fields: datatypes.JSON(fields), RequiredFields: datatypes.JSON(requiredFields), SkuRules: datatypes.JSON(skuRules), DetailTemplate: datatypes.JSON(detailTemplate), IsActive: isActive}
	if err := s.listing1688CategoryTemplateRepo().Upsert(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": save category template failed", err)
		return
	}
	s.response.Success(ctx, row)
}

func (s *Service) Listing1688PublishFormValidate(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	var req listing1688PublishFormValidateReq
	_ = ctx.ShouldBindJSON(&req)
	categoryID := listing1688ResolveMatchedCategoryID(workflow, req.CategoryID)
	if categoryID == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": category_id is required", nil)
		return
	}
	storeKey := listing1688StoreKey(req.StoreKey)
	userID := workflow.UserID
	template, err := s.listing1688CategoryTemplateRepo().GetActiveByUserStoreCategory(userID, storeKey, categoryID)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": active category template not found", err)
		return
	}
	presetFields := map[string]any{}
	if preset, presetErr := s.listing1688StorePresetRepo().GetByUserStore(userID, storeKey); presetErr == nil && preset.IsActive {
		presetFields = listing1688DecodePublishFieldsFromJSON(preset.Fields)
	}
	images, title := s.listing1688SelectedImagesAndTitle(workflow)
	draft := listing1688BuildPublishDraft(listing1688PublishProfileInput{Workflow: listing1688PublishWorkflowData{Title: title, Images: images, Skus: listing1688SnapshotSkus(workflow.OfferSnapshot)}, StorePreset: listing1688PublishStorePreset{Fields: presetFields}, CategoryTemplate: listing1688PublishCategoryTemplate{CategoryID: template.CategoryID, Fields: listing1688DecodePublishFieldsFromJSON(template.Fields), RequiredFields: listing1688DecodePublishRequiredFields(template.RequiredFields)}, UserFields: req.Fields})
	formData, _ := json.Marshal(draft.FormData)
	missingFields, _ := json.Marshal(draft.MissingFields)
	row := &modules.TableListing1688PublishDrafts{WorkflowID: workflow.ID, UserID: userID, StoreKey: storeKey, CategoryID: draft.CategoryID, FormData: datatypes.JSON(formData), MissingFields: datatypes.JSON(missingFields), ValidationStatus: draft.ValidationStatus, DraftSaved: false}
	if err := s.listing1688PublishDraftRepo().Upsert(row); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": save publish draft failed", err)
		return
	}
	s.response.Success(ctx, gin.H{"draft_id": row.ID, "category_id": draft.CategoryID, "form_data": draft.FormData, "missing_fields": draft.MissingFields, "validation_status": draft.ValidationStatus, "draft_saved": false})
}

type listing1688SellerSaveDraftResp struct {
	DraftSaved       bool     `json:"draft_saved"`
	ValidationErrors []string `json:"validation_errors"`
	PageURL          string   `json:"page_url"`
	TitleFilled      bool     `json:"title_filled"`
	ImagesSet        int      `json:"images_set"`
}

// Listing1688PublishFormSaveDraft 同步调用 Agent 保存草稿，成功后回写 draft_saved
func (s *Service) Listing1688PublishFormSaveDraft(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if err := s.listing1688AssertImgReady(workflow); err != nil {
		s.response.Forbidden(ctx, err.Error(), err)
		return
	}
	if !s.listing1688RequireOnlineAgent(workflow.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	if _, err := s.listing1688RequireSellerSession(workflow); err != nil {
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
	draft, err := s.listing1688PublishDraftRepo().GetByWorkflow(workflow.ID)
	if err != nil || draft == nil || draft.ValidationStatus != listing1688PublishDraftStatusReady {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "请先完成类目表单校验"), err)
		return
	}
	formData := listing1688DecodePublishFieldsFromJSON(draft.FormData)
	title, _ := formData["title"].(string)
	title = strings.TrimSpace(title)
	images, _ := s.listing1688SelectedImagesAndTitle(workflow)
	if title == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "发布草案标题为空"), nil)
		return
	}
	if len(images) < 1 {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrImagesNotReady, ""), nil)
		return
	}
	send := map[string]any{
		"confirm_save_draft": true,
		"category_id":        draft.CategoryID,
		"publish_form":       formData,
		"title":              title,
		"images":             images,
	}
	resp, syncErr := s.listing1688SyncAgent(workflow.AgentID, constants.ProtocolListing1688SellerSaveDraft, send, 60*time.Second)
	if syncErr != nil || resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "save draft failed"
		if syncErr != nil {
			detail = syncErr.Error()
		} else if resp != nil && strings.TrimSpace(resp.Msg) != "" {
			detail = resp.Msg
		}
		draft.DraftSaved = false
		_ = s.listing1688PublishDraftRepo().Upsert(draft)
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, detail), syncErr)
		return
	}
	var saved listing1688SellerSaveDraftResp
	_ = json.Unmarshal(resp.Data, &saved)
	if !saved.DraftSaved {
		draft.DraftSaved = false
		_ = s.listing1688PublishDraftRepo().Upsert(draft)
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "Agent 未确认草稿已保存"), nil)
		return
	}
	draft.DraftSaved = true
	if err := s.listing1688PublishDraftRepo().Upsert(draft); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": update draft_saved failed", err)
		return
	}
	s.response.Success(ctx, gin.H{
		"draft_id":          draft.ID,
		"category_id":       draft.CategoryID,
		"draft_saved":       true,
		"validation_status": draft.ValidationStatus,
		"page_url":          saved.PageURL,
		"validation_errors": saved.ValidationErrors,
		"title_filled":      saved.TitleFilled,
		"images_set":        saved.ImagesSet,
	})
}

func listing1688DecodePublishFieldsFromJSON(raw datatypes.JSON) map[string]any {
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	if fields == nil {
		return map[string]any{}
	}
	return fields
}
func listing1688DecodePublishRequiredFields(raw datatypes.JSON) []string {
	var fields []string
	_ = json.Unmarshal(raw, &fields)
	return fields
}

type listing1688CategoryInspectReq struct {
	StoreKey      string `json:"store_key"`
	CategoryID    string `json:"category_id"`
	CategoryPath  string `json:"category_path"`
	ProductFamily string `json:"product_family"`
}

type listing1688SellerCategoryInspectResp struct {
	CategoryID       string            `json:"category_id"`
	PageURL          string            `json:"page_url"`
	RequiredFields   []string          `json:"required_fields"`
	ValidationErrors []string          `json:"validation_errors"`
	SaveDraftReady   bool              `json:"save_draft_ready"`
	SubmitReady      bool              `json:"submit_ready"`
	FieldBindings    map[string]string `json:"field_bindings"`
}

func (s *Service) Listing1688CategoryInspect(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	var req listing1688CategoryInspectReq
	_ = ctx.ShouldBindJSON(&req)
	categoryID := strings.TrimSpace(req.CategoryID)
	if categoryID == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": category_id is required", nil)
		return
	}
	if _, err := s.listing1688RequireSellerSession(workflow); err != nil {
		code := constants.ParseListing1688WizardErrorCode(err.Error())
		if code == "" {
			code = constants.Listing1688ErrSellerSessionInvalid
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(code, ""), err)
		return
	}
	resp, err := s.listing1688SyncAgent(workflow.AgentID, constants.ProtocolListing1688SellerInspectCategory, map[string]any{"category_id": categoryID}, 30*time.Second)
	if err != nil || resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "category form inspect failed"
		if err != nil {
			detail = err.Error()
		} else if resp != nil && strings.TrimSpace(resp.Msg) != "" {
			detail = resp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, detail), err)
		return
	}
	var inspected listing1688SellerCategoryInspectResp
	_ = json.Unmarshal(resp.Data, &inspected)
	if strings.TrimSpace(inspected.CategoryID) != "" && strings.TrimSpace(inspected.CategoryID) != categoryID {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": inspected category does not match request", nil)
		return
	}
	if len(inspected.RequiredFields) == 0 || !inspected.SaveDraftReady {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": seller form inspection is incomplete", nil)
		return
	}
	storeKey := listing1688StoreKey(req.StoreKey)
	existing, _ := s.listing1688CategoryTemplateRepo().GetActiveByUserStoreCategory(workflow.UserID, storeKey, categoryID)
	fields := datatypes.JSON([]byte(`{}`))
	skuRules := datatypes.JSON([]byte(`{}`))
	detailTemplate := datatypes.JSON([]byte(`{}`))
	categoryPath := strings.TrimSpace(req.CategoryPath)
	productFamily := strings.TrimSpace(req.ProductFamily)
	if existing != nil {
		fields = existing.Fields
		skuRules = existing.SkuRules
		detailTemplate = existing.DetailTemplate
		if categoryPath == "" {
			categoryPath = existing.CategoryPath
		}
		if productFamily == "" {
			productFamily = existing.ProductFamily
		}
	}
	requiredFields := listing1688ResolvedTemplateRequiredFields(listing1688DecodePublishRequiredFields(func() datatypes.JSON {
		if existing != nil {
			return existing.RequiredFields
		}
		return nil
	}()), inspected.RequiredFields)
	rawRequiredFields, _ := json.Marshal(requiredFields)
	template := &modules.TableListing1688CategoryTemplates{UserID: workflow.UserID, StoreKey: storeKey, CategoryID: categoryID, CategoryPath: categoryPath, ProductFamily: productFamily, Fields: fields, RequiredFields: datatypes.JSON(rawRequiredFields), SkuRules: skuRules, DetailTemplate: detailTemplate, IsActive: true}
	if err := s.listing1688CategoryTemplateRepo().Upsert(template); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": save inspected category template failed", err)
		return
	}
	s.response.Success(ctx, gin.H{"template_id": template.ID, "category_id": categoryID, "required_fields": requiredFields, "validation_errors": inspected.ValidationErrors, "save_draft_ready": inspected.SaveDraftReady, "submit_ready": inspected.SubmitReady, "page_url": inspected.PageURL})
}

func listing1688ResolvedTemplateRequiredFields(existing, observed []string) []string {
	if len(observed) == 0 {
		return existing
	}
	out := make([]string, 0, len(observed))
	seen := make(map[string]struct{})
	for _, field := range observed {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}
