package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Listing1688SellerSessionOpenLogin Step4：打开商家登录/工作台页（不关页，类店小秘）
func (s *Service) Listing1688SellerSessionOpenLogin(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	resp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688SellerOpenLogin, nil, 45*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, err.Error()), err)
		return
	}
	if resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "打开商家登录页失败"
		if resp != nil && resp.Msg != "" {
			detail = resp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, detail), nil)
		return
	}
	s.response.Success(ctx, gin.H{
		"workflow_id": row.ID,
		"status":      row.Status,
		"agent":       json.RawMessage(resp.Data),
		"hint":        "请在 Agent 浏览器登录 1688 商家中心后点「我已登录商家中心」",
	})
}

// Listing1688SellerSessionConfirm 导出商家 Cookie → 加密 → 商家探针
func (s *Service) Listing1688SellerSessionConfirm(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	exportResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688SellerExportCookie, nil, 45*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, err.Error()), err)
		return
	}
	if exportResp == nil || (exportResp.Code != 0 && exportResp.Code != 200) {
		detail := "导出商家 Cookie 失败"
		if exportResp != nil && exportResp.Msg != "" {
			detail = exportResp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, detail), nil)
		return
	}
	var exportData struct {
		Cookie string `json:"cookie"`
	}
	_ = json.Unmarshal(exportResp.Data, &exportData)
	cookie := strings.TrimSpace(exportData.Cookie)
	if cookie == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, "Cookie 为空"), nil)
		return
	}
	enc, err := utils.Listing1688EncryptCookie(cookie)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": Cookie 加密失败: "+err.Error(), err)
		return
	}
	if s.logger != nil {
		s.logger.Info("listing1688 seller confirm: cookie=%s workflow=%d", utils.Listing1688RedactCookiePlain(cookie), row.ID)
	}

	probeResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688SellerProbe, map[string]any{"cookie": cookie}, 30*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, err.Error()), err)
		return
	}
	probeOK := probeResp != nil && (probeResp.Code == 0 || probeResp.Code == 200)
	probeMsg := ""
	if probeResp != nil {
		probeMsg = probeResp.Msg
	}
	now := time.Now()
	sess := &modules.TableListing1688Sessions{
		WorkflowID:      row.ID,
		UserID:          row.UserID,
		SessionType:     constants.Listing1688SessionSeller,
		CookieEnc:       enc,
		ProbeOK:         probeOK,
		ProbeMessage:    probeMsg,
		LastValidatedAt: &now,
	}
	if err := s.listing1688Sessions().UpsertByWorkflowType(sess); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 保存商家会话失败", err)
		return
	}
	if !probeOK {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, probeMsg), nil)
		return
	}
	s.response.Success(ctx, gin.H{
		"workflow_id": row.ID,
		"seller_ok":   true,
		// 永不回传 Cookie 明文
	})
}

// Listing1688SellerSessionValidate 重新探针商家会话
func (s *Service) Listing1688SellerSessionValidate(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	sess, err := s.listing1688Sessions().GetLatestByWorkflow(row.ID, constants.Listing1688SessionSeller)
	if err != nil || sess == nil || sess.CookieEnc == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, "无商家会话"), err)
		return
	}
	plain, err := utils.Listing1688DecryptCookie(sess.CookieEnc)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": Cookie 解密失败", err)
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	probeResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688SellerProbe, map[string]any{"cookie": plain}, 30*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, err.Error()), err)
		return
	}
	probeOK := probeResp != nil && (probeResp.Code == 0 || probeResp.Code == 200)
	probeMsg := ""
	if probeResp != nil {
		probeMsg = probeResp.Msg
	}
	now := time.Now()
	_ = s.listing1688Sessions().Update(sess.ID, map[string]interface{}{
		"probe_ok":          probeOK,
		"probe_message":     probeMsg,
		"last_validated_at": now,
	})
	if !probeOK {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSellerSessionInvalid, probeMsg), nil)
		return
	}
	s.response.Success(ctx, gin.H{"workflow_id": row.ID, "seller_ok": true})
}

func (s *Service) listing1688RequireSellerSession(row *modules.TableListing1688Workflows) (string, error) {
	sess, err := s.listing1688Sessions().GetLatestByWorkflow(row.ID, constants.Listing1688SessionSeller)
	if err != nil || sess == nil || !sess.ProbeOK || sess.CookieEnc == "" {
		return "", fmt.Errorf("%s", constants.Listing1688ErrSellerSessionInvalid)
	}
	plain, err := utils.Listing1688DecryptCookie(sess.CookieEnc)
	if err != nil {
		return "", err
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		return "", fmt.Errorf("%s", constants.Listing1688ErrWizardAgentOffline)
	}
	probeResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688SellerProbe, map[string]any{"cookie": plain}, 30*time.Second)
	if err != nil {
		return "", err
	}
	if probeResp == nil || (probeResp.Code != 0 && probeResp.Code != 200) {
		return "", fmt.Errorf("%s", constants.Listing1688ErrSellerSessionInvalid)
	}
	return plain, nil
}
