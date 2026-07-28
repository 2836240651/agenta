package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type listing1688CollectReq struct {
	URL string `json:"url"`
}

type listing1688OfferSnapshot struct {
	OK         bool            `json:"ok"`
	Status     string          `json:"status"`
	URL        string          `json:"url"`
	OfferID    string          `json:"source_item_id"`
	Title      string          `json:"title"`
	MainImages []string        `json:"main_images"`
	Skus       json.RawMessage `json:"skus"`
	PriceMin   *float64        `json:"price_min"`
	PriceMax   *float64        `json:"price_max"`
	Error      string          `json:"error"`
	Captcha    bool            `json:"captcha"`
	NeedLogin  bool            `json:"need_login"`
}

func (s *Service) listing1688DoCollect(ctx *gin.Context, isRetry bool) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if row.Status != string(constants.Listing1688StatusSessionValid) &&
		row.Status != string(constants.Listing1688StatusCollected) &&
		row.Status != string(constants.Listing1688StatusFailed) &&
		row.Status != string(constants.Listing1688StatusCollecting) {
		if !isRetry {
			s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrCollectNeedLogin, ""), nil)
			return
		}
	}
	var req listing1688CollectReq
	_ = ctx.ShouldBindJSON(&req)
	offerURL := strings.TrimSpace(req.URL)
	if offerURL == "" {
		offerURL = strings.TrimSpace(row.OfferURL)
	}
	if offerURL == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrCollectParseFail, "请提供竞品链接"), nil)
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}

	sess, err := s.listing1688SessionRepo().GetLatestByWorkflow(row.ID, constants.Listing1688SessionCollect)
	cookiePlain := ""
	if err == nil && sess != nil && sess.CookieEnc != "" {
		cookiePlain, _ = utils.Listing1688DecryptCookie(sess.CookieEnc)
	}

	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":    string(constants.Listing1688StatusCollecting),
		"offer_url": offerURL,
	})

	send := map[string]any{"url": offerURL}
	if cookiePlain != "" {
		send["cookie"] = cookiePlain
	}
	resp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688FetchOffer, send, 120*time.Second)
	if err != nil {
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(constants.Listing1688ErrWizardAgentOffline),
			"message":    err.Error(),
		})
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, err.Error()), err)
		return
	}
	if resp == nil || (resp.Code != 0 && resp.Code != 200) {
		code := constants.Listing1688ErrCollectParseFail
		detail := "采集失败"
		if resp != nil && resp.Msg != "" {
			detail = resp.Msg
			if strings.Contains(detail, "COLLECT_CAPTCHA") || respCaptcha(detail) {
				code = constants.Listing1688ErrCollectCaptcha
			} else if strings.Contains(detail, "COLLECT_NEED_LOGIN") || respNeedLogin(detail) {
				code = constants.Listing1688ErrCollectNeedLogin
			}
		}
		msg := constants.FormatListing1688WizardError(code, detail)
		// 保持 session_valid，便于验证码/解析失败后重试（不锁死 Step2）
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(code),
			"message":    msg,
		})
		s.response.Forbidden(ctx, msg, nil)
		return
	}

	var snap listing1688OfferSnapshot
	_ = json.Unmarshal(resp.Data, &snap)
	if snap.Captcha {
		msg := constants.FormatListing1688WizardError(constants.Listing1688ErrCollectCaptcha, "")
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(constants.Listing1688ErrCollectCaptcha),
			"message":    msg,
		})
		s.response.Forbidden(ctx, msg, nil)
		return
	}
	if snap.NeedLogin {
		msg := constants.FormatListing1688WizardError(constants.Listing1688ErrCollectNeedLogin, "")
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(constants.Listing1688ErrCollectNeedLogin),
			"message":    msg,
		})
		s.response.Forbidden(ctx, msg, nil)
		return
	}
	if !snap.OK || strings.TrimSpace(snap.Title) == "" || len(snap.MainImages) < 1 {
		detail := "未达采集门槛（标题+图片+SKU）"
		if snap.Error != "" {
			detail = snap.Error
		}
		msg := constants.FormatListing1688WizardError(constants.Listing1688ErrCollectParseFail, detail)
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(constants.Listing1688ErrCollectParseFail),
			"message":    msg,
		})
		s.response.Forbidden(ctx, msg, nil)
		return
	}
	var skus []any
	_ = json.Unmarshal(snap.Skus, &skus)
	if len(skus) < 1 {
		msg := constants.FormatListing1688WizardError(constants.Listing1688ErrCollectParseFail, "缺少 SKU")
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusSessionValid),
			"error_code": string(constants.Listing1688ErrCollectParseFail),
			"message":    msg,
		})
		s.response.Forbidden(ctx, msg, nil)
		return
	}

	rawSnap, _ := json.Marshal(snap)
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":         string(constants.Listing1688StatusCollected),
		"offer_url":      offerURL,
		"offer_snapshot": datatypes.JSON(rawSnap),
		"error_code":     "",
		"message":        "",
	})
	s.response.Success(ctx, gin.H{
		"workflow_id": row.ID,
		"status":      constants.Listing1688StatusCollected,
		"preview": gin.H{
			"title":       snap.Title,
			"main_images": snap.MainImages,
			"skus":        json.RawMessage(snap.Skus),
			"price_min":   snap.PriceMin,
			"price_max":   snap.PriceMax,
			"offer_id":    snap.OfferID,
			"url":         snap.URL,
		},
	})
}

func respCaptcha(msg string) bool {
	return strings.Contains(strings.ToLower(msg), "captcha") || strings.Contains(msg, "验证码")
}

func respNeedLogin(msg string) bool {
	// 仅认明确会话失效码/文案；勿匹配「可能改版或需登录态」等解析失败提示里的「登录」
	if strings.Contains(msg, "COLLECT_NEED_LOGIN") {
		return true
	}
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "need_login") || strings.Contains(lower, "need login") {
		return true
	}
	return strings.Contains(msg, "需要重新登录") || strings.Contains(msg, "会话失效") || strings.Contains(msg, "未登录")
}

func (s *Service) Listing1688Collect(ctx *gin.Context) {
	s.listing1688DoCollect(ctx, false)
}

func (s *Service) Listing1688CollectRetry(ctx *gin.Context) {
	s.listing1688DoCollect(ctx, true)
}
