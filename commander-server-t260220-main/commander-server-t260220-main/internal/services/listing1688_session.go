package services

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/modules"
	"commander-server-t260220/internal/pkg"
	"commander-server-t260220/internal/repos"
	"commander-server-t260220/internal/types"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type listing1688AgentResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type listing1688SessionStore interface {
	Create(row *modules.TableListing1688Sessions) error
	GetLatestByWorkflow(workflowID uint, sessionType string) (*modules.TableListing1688Sessions, error)
	Update(id uint, updates map[string]interface{}) error
	UpsertByWorkflowType(row *modules.TableListing1688Sessions) error
}

func (s *Service) listing1688Sessions() listing1688SessionStore {
	if s.testListing1688Sessions != nil {
		return s.testListing1688Sessions
	}
	return repos.NewListing1688SessionRepo(s.gormUtils.Get())
}

// listing1688SessionRepo 兼容旧调用
func (s *Service) listing1688SessionRepo() listing1688SessionStore {
	return s.listing1688Sessions()
}

func (s *Service) listing1688SyncAgent(agentID, protocol string, send any, timeout time.Duration) (*listing1688AgentResp, error) {
	taskID := pkg.TextGenerateRandomString()
	msg := types.AgentMessage[any, any]{
		TaskId:   taskID,
		AgentId:  agentID,
		Platform: constants.Platform1688,
		Protocol: protocol,
		Send:     send,
	}
	raw, err := s.agentManager.SyncSend(agentID, msg, timeout)
	if err != nil {
		return nil, err
	}
	var envelope types.AgentMessage[any, listing1688AgentResp]
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	return &envelope.Receive, nil
}

func (s *Service) listing1688RequireOnlineAgent(agentID string) bool {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" || s.agentManager == nil {
		return false
	}
	for _, a := range s.agentManager.GetAllAgents() {
		if a != nil && strings.EqualFold(strings.TrimSpace(a.ID), agentID) {
			return true
		}
	}
	return false
}

// Listing1688SessionOpenLogin Step1：打开登录页 → session_pending
func (s *Service) Listing1688SessionOpenLogin(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	resp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688OpenLogin, nil, 45*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, err.Error()), err)
		return
	}
	if resp == nil || (resp.Code != 0 && resp.Code != 200) {
		detail := "打开登录页失败"
		if resp != nil && resp.Msg != "" {
			detail = resp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, detail), nil)
		return
	}
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":     string(constants.Listing1688StatusSessionPending),
		"error_code": "",
		"message":    "",
	})
	s.response.Success(ctx, gin.H{
		"workflow_id": row.ID,
		"status":      constants.Listing1688StatusSessionPending,
		"agent":       json.RawMessage(resp.Data),
	})
}

// Listing1688SessionConfirm 我已登录 → 导出 Cookie 加密落库 → 探针 → session_valid
func (s *Service) Listing1688SessionConfirm(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	if !s.listing1688RequireOnlineAgent(row.AgentID) {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, ""), nil)
		return
	}
	exportResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688ExportCookie, nil, 45*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrWizardAgentOffline, err.Error()), err)
		return
	}
	if exportResp == nil || (exportResp.Code != 0 && exportResp.Code != 200) {
		detail := "导出 Cookie 失败"
		if exportResp != nil && exportResp.Msg != "" {
			detail = exportResp.Msg
		}
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrCollectNeedLogin, detail), nil)
		return
	}
	var exportData struct {
		Cookie string `json:"cookie"`
	}
	_ = json.Unmarshal(exportResp.Data, &exportData)
	cookie := strings.TrimSpace(exportData.Cookie)
	if cookie == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrCollectNeedLogin, "Cookie 为空"), nil)
		return
	}
	enc, err := utils.Listing1688EncryptCookie(cookie)
	if err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": Cookie 加密失败（检查 LISTING_1688_COOKIE_KEY）: "+err.Error(), err)
		return
	}
	if s.logger != nil {
		s.logger.Info("listing1688 session confirm: cookie=%s workflow=%d", utils.Listing1688RedactCookiePlain(cookie), row.ID)
	}

	probeResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688Probe, map[string]any{"cookie": cookie}, 30*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, err.Error()), err)
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
		SessionType:     constants.Listing1688SessionCollect,
		CookieEnc:       enc,
		ProbeOK:         probeOK,
		ProbeMessage:    probeMsg,
		LastValidatedAt: &now,
	}
	if err := s.listing1688SessionRepo().Create(sess); err != nil {
		s.response.Forbidden(ctx, string(constants.Listing1688ErrInternal)+": 保存会话失败", err)
		return
	}
	if !probeOK {
		_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusFailed),
			"error_code": string(constants.Listing1688ErrSessionInvalid),
			"message":    probeMsg,
		})
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, probeMsg), nil)
		return
	}
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status":     string(constants.Listing1688StatusSessionValid),
		"error_code": "",
		"message":    "",
	})
	s.response.Success(ctx, gin.H{
		"workflow_id": row.ID,
		"status":      constants.Listing1688StatusSessionValid,
		"probe_ok":    true,
		// 永不回传 Cookie 明文
	})
}

// Listing1688SessionValidate 重新探针（使用已存加密 Cookie）
func (s *Service) Listing1688SessionValidate(ctx *gin.Context) {
	row, ok := s.listing1688LoadOwnedWorkflow(ctx)
	if !ok {
		return
	}
	sess, err := s.listing1688SessionRepo().GetLatestByWorkflow(row.ID, constants.Listing1688SessionCollect)
	if err != nil || sess == nil || sess.CookieEnc == "" {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, "无采集会话"), err)
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
	probeResp, err := s.listing1688SyncAgent(row.AgentID, constants.ProtocolListing1688Probe, map[string]any{"cookie": plain}, 30*time.Second)
	if err != nil {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, err.Error()), err)
		return
	}
	probeOK := probeResp != nil && (probeResp.Code == 0 || probeResp.Code == 200)
	probeMsg := ""
	if probeResp != nil {
		probeMsg = probeResp.Msg
	}
	now := time.Now()
	_ = s.listing1688SessionRepo().Update(sess.ID, map[string]interface{}{
		"probe_ok":          probeOK,
		"probe_message":     probeMsg,
		"last_validated_at": now,
	})
	if !probeOK {
		s.response.Forbidden(ctx, constants.FormatListing1688WizardError(constants.Listing1688ErrSessionInvalid, probeMsg), nil)
		return
	}
	_ = s.listing1688Workflows().Update(row.ID, map[string]interface{}{
		"status": string(constants.Listing1688StatusSessionValid),
	})
	s.response.Success(ctx, gin.H{"workflow_id": row.ID, "status": constants.Listing1688StatusSessionValid, "probe_ok": true})
}
