package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

const (
	listing1688MatchStatusPending     = "pending"
	listing1688MatchStatusRecommended = "recommended"
	listing1688MatchStatusSelected    = "selected"
	listing1688MatchStatusOpened      = "opened"
)

type listing1688TemplateProductsResp struct {
	Products []listing1688TemplateProduct `json:"products"`
	PageURL  string                       `json:"page_url"`
}

type listing1688TemplateProduct struct {
	OfferID    string   `json:"offer_id"`
	Title      string   `json:"title"`
	CategoryID string   `json:"category_id,omitempty"`
	PriceText  string   `json:"price_text,omitempty"`
	PriceMin   *float64 `json:"price_min,omitempty"`
	PriceMax   *float64 `json:"price_max,omitempty"`
}

type listing1688TemplateCandidate struct {
	OfferID        string         `json:"offer_id"`
	Title          string         `json:"title"`
	CategoryID     string         `json:"category_id,omitempty"`
	PriceText      string         `json:"price_text,omitempty"`
	PriceMin       *float64       `json:"price_min,omitempty"`
	PriceMax       *float64       `json:"price_max,omitempty"`
	Score          int            `json:"score"`
	ScoreBreakdown map[string]int `json:"score_breakdown"`
	ReasonSummary  string         `json:"reason_summary"`
}

type listing1688TemplateCandidatesReq struct {
	Limit int `json:"limit"`
}

type listing1688TemplateSelectReq struct {
	OfferID    string `json:"offer_id"`
	Title      string `json:"title"`
	CategoryID string `json:"category_id"`
}

type listing1688TemplateOpenSimilarReq struct {
	OfferID string `json:"offer_id"`
}

type listing1688OpenSimilarResp struct {
	OfferID    string `json:"offer_id"`
	CategoryID string `json:"category_id"`
	PageURL    string `json:"page_url"`
}

func (s *Service) Listing1688TemplateCandidates(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(workflow.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
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

	var req listing1688TemplateCandidatesReq
	_ = ctx.ShouldBindJSON(&req)
	limit := req.Limit
	if limit <= 0 || limit > 10 {
		limit = 3
	}

	resp, err := s.listing1688SyncAgent(workflow.AgentID, constants.ProtocolListing1688SellerListProducts, map[string]any{"limit": 120}, 60*time.Second)
	if err != nil || resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "读取店内商品候选失败"
		if err != nil {
			detail = err.Error()
		} else if resp != nil && strings.TrimSpace(resp.Msg) != "" {
			detail = resp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, detail), err)
		return
	}

	var productsResp listing1688TemplateProductsResp
	_ = json.Unmarshal(resp.Data, &productsResp)
	candidates := listing1688RankTemplateCandidates(workflow, productsResp.Products, limit)
	status := listing1688MatchStatusPending
	if len(candidates) > 0 {
		status = listing1688MatchStatusRecommended
	}
	rawCandidates, _ := json.Marshal(candidates)
	_ = s.listing1688Workflows().Update(workflow.ID, map[string]interface{}{
		"match_candidates": datatypes.JSON(rawCandidates),
		"match_status":     status,
	})
	s.response.Success(ctx, gin.H{
		"workflow_id":  workflow.ID,
		"match_status": status,
		"page_url":     productsResp.PageURL,
		"candidates":   candidates,
	})
}

func (s *Service) Listing1688TemplateSelect(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	var req listing1688TemplateSelectReq
	_ = ctx.ShouldBindJSON(&req)
	offerID := strings.TrimSpace(req.OfferID)
	if offerID == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": offer_id is required", nil)
		return
	}
	updates := map[string]interface{}{
		"matched_template_offer_id": offerID,
		"match_status":              listing1688MatchStatusSelected,
	}
	if categoryID := strings.TrimSpace(req.CategoryID); categoryID != "" {
		updates["matched_category_id"] = categoryID
	}
	if err := s.listing1688Workflows().Update(workflow.ID, updates); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": save template selection failed", err)
		return
	}
	s.response.Success(ctx, gin.H{
		"workflow_id":               workflow.ID,
		"matched_template_offer_id": offerID,
		"matched_category_id":       strings.TrimSpace(req.CategoryID),
		"match_status":              listing1688MatchStatusSelected,
	})
}

func (s *Service) Listing1688TemplateOpenSimilar(ctx *gin.Context) {
	workflow, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(workflow.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
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

	var req listing1688TemplateOpenSimilarReq
	_ = ctx.ShouldBindJSON(&req)
	offerID := strings.TrimSpace(req.OfferID)
	if offerID == "" {
		offerID = strings.TrimSpace(workflow.MatchedTemplateOfferID)
	}
	if offerID == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": template offer is required", nil)
		return
	}

	resp, err := s.listing1688SyncAgent(workflow.AgentID, constants.ProtocolListing1688SellerOpenSimilarPublish, map[string]any{"offer_id": offerID}, 60*time.Second)
	if err != nil || resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "打开发布相似品页失败"
		if err != nil {
			detail = err.Error()
		} else if resp != nil && strings.TrimSpace(resp.Msg) != "" {
			detail = resp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, detail), err)
		return
	}

	var opened listing1688OpenSimilarResp
	_ = json.Unmarshal(resp.Data, &opened)
	if strings.TrimSpace(opened.CategoryID) == "" {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrWizardPublishFail)+": category_id missing after open similar", nil)
		return
	}
	if err := s.listing1688Workflows().Update(workflow.ID, map[string]interface{}{
		"matched_template_offer_id": offerID,
		"matched_category_id":       strings.TrimSpace(opened.CategoryID),
		"similar_page_url":          strings.TrimSpace(opened.PageURL),
		"match_status":              listing1688MatchStatusOpened,
	}); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": update similar page result failed", err)
		return
	}
	s.response.Success(ctx, gin.H{
		"workflow_id":               workflow.ID,
		"matched_template_offer_id": offerID,
		"matched_category_id":       strings.TrimSpace(opened.CategoryID),
		"similar_page_url":          strings.TrimSpace(opened.PageURL),
		"match_status":              listing1688MatchStatusOpened,
	})
}

func listing1688RankTemplateCandidates(workflow *modules.TableListing1688Workflows, products []listing1688TemplateProduct, limit int) []listing1688TemplateCandidate {
	if workflow == nil || limit <= 0 {
		return []listing1688TemplateCandidate{}
	}
	title := ""
	var priceMin, priceMax *float64
	sourceSKUCount := len(listing1688SnapshotSkus(workflow.OfferSnapshot))
	if len(workflow.OfferSnapshot) > 0 {
		var snap map[string]any
		_ = json.Unmarshal(workflow.OfferSnapshot, &snap)
		if v, ok := snap["title"].(string); ok {
			title = strings.TrimSpace(v)
		}
		priceMin = listing1688FloatPtr(snap["price_min"])
		priceMax = listing1688FloatPtr(snap["price_max"])
	}
	titleGrams := listing1688TitleGrams(title)
	scored := make([]listing1688TemplateCandidate, 0, len(products))
	for _, product := range products {
		offerID := strings.TrimSpace(product.OfferID)
		productTitle := strings.TrimSpace(product.Title)
		if offerID == "" || productTitle == "" {
			continue
		}
		breakdown := map[string]int{
			"title": listing1688TitleScore(titleGrams, listing1688TitleGrams(productTitle)),
			"price": listing1688PriceScore(priceMin, priceMax, product.PriceMin, product.PriceMax),
			"sku":   listing1688SKUShapeScore(sourceSKUCount, 1),
		}
		score := breakdown["title"] + breakdown["price"] + breakdown["sku"]
		scored = append(scored, listing1688TemplateCandidate{
			OfferID:        offerID,
			Title:          productTitle,
			CategoryID:     strings.TrimSpace(product.CategoryID),
			PriceText:      strings.TrimSpace(product.PriceText),
			PriceMin:       product.PriceMin,
			PriceMax:       product.PriceMax,
			Score:          score,
			ScoreBreakdown: breakdown,
			ReasonSummary:  listing1688CandidateReasonSummary(breakdown),
		})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].OfferID < scored[j].OfferID
	})
	if len(scored) > limit {
		scored = scored[:limit]
	}
	return scored
}

func listing1688TitleGrams(title string) map[string]struct{} {
	out := make(map[string]struct{})
	normalized := listing1688NormalizeTitle(title)
	if normalized == "" {
		return out
	}
	runes := []rune(normalized)
	if len(runes) == 1 {
		out[string(runes[0])] = struct{}{}
		return out
	}
	for i := 0; i < len(runes)-1; i++ {
		out[string(runes[i:i+2])] = struct{}{}
	}
	return out
}

func listing1688NormalizeTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[\p{P}\p{S}\s]+`)
	return re.ReplaceAllString(title, "")
}

func listing1688TitleScore(source, candidate map[string]struct{}) int {
	if len(source) == 0 || len(candidate) == 0 {
		return 0
	}
	intersect := 0
	for key := range source {
		if _, ok := candidate[key]; ok {
			intersect++
		}
	}
	union := len(source) + len(candidate) - intersect
	if union <= 0 {
		return 0
	}
	return int(math.Round((float64(intersect) / float64(union)) * 20))
}

func listing1688PriceScore(sourceMin, sourceMax, candidateMin, candidateMax *float64) int {
	if sourceMin == nil || candidateMin == nil {
		return 0
	}
	sMid := *sourceMin
	if sourceMax != nil {
		sMid = (sMid + *sourceMax) / 2
	}
	cMid := *candidateMin
	if candidateMax != nil {
		cMid = (cMid + *candidateMax) / 2
	}
	if sMid <= 0 || cMid <= 0 {
		return 0
	}
	diff := math.Abs(sMid-cMid) / math.Max(sMid, cMid)
	switch {
	case diff <= 0.1:
		return 10
	case diff <= 0.25:
		return 7
	case diff <= 0.5:
		return 4
	default:
		return 1
	}
}

func listing1688SKUShapeScore(sourceCount, candidateCount int) int {
	if sourceCount <= 0 || candidateCount <= 0 {
		return 0
	}
	sourceBucket := listing1688SKUBucket(sourceCount)
	candidateBucket := listing1688SKUBucket(candidateCount)
	if sourceBucket == candidateBucket {
		return 10
	}
	if absInt(sourceBucket-candidateBucket) == 1 {
		return 5
	}
	return 1
}

func listing1688SKUBucket(count int) int {
	switch {
	case count <= 1:
		return 1
	case count <= 5:
		return 2
	default:
		return 3
	}
}

func listing1688CandidateReasonSummary(breakdown map[string]int) string {
	parts := make([]string, 0, 3)
	if breakdown["title"] > 0 {
		parts = append(parts, "标题接近")
	}
	if breakdown["price"] >= 7 {
		parts = append(parts, "价格带接近")
	}
	if breakdown["sku"] >= 5 {
		parts = append(parts, "规格形态接近")
	}
	if len(parts) == 0 {
		return "基础候选"
	}
	return strings.Join(parts, " / ")
}

func listing1688FloatPtr(v any) *float64 {
	switch typed := v.(type) {
	case float64:
		return &typed
	case float32:
		f := float64(typed)
		return &f
	case int:
		f := float64(typed)
		return &f
	case int64:
		f := float64(typed)
		return &f
	case json.Number:
		f, err := typed.Float64()
		if err == nil {
			return &f
		}
	case string:
		s := strings.TrimSpace(typed)
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err == nil {
			return &f
		}
	}
	return nil
}

func listing1688WorkflowTemplateMatchResult(row *modules.TableListing1688Workflows) gin.H {
	candidates := make([]listing1688TemplateCandidate, 0)
	if row != nil && len(row.MatchCandidates) > 0 {
		_ = json.Unmarshal(row.MatchCandidates, &candidates)
	}
	return gin.H{
		"status":               strings.TrimSpace(row.MatchStatus),
		"selected_offer_id":    strings.TrimSpace(row.MatchedTemplateOfferID),
		"selected_category_id": strings.TrimSpace(row.MatchedCategoryID),
		"similar_page_url":     strings.TrimSpace(row.SimilarPageURL),
		"candidates":           candidates,
	}
}

func listing1688ResolveMatchedCategoryID(row *modules.TableListing1688Workflows, requested string) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	if row == nil {
		return ""
	}
	return strings.TrimSpace(row.MatchedCategoryID)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
