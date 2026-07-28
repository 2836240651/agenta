package tasks

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/manager"
	"commander-server-t260220/internal/pkg"
	"commander-server-t260220/internal/types"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type listing1688SellerPublishResult struct {
	OK          bool   `json:"ok"`
	OfferID     string `json:"offer_id"`
	PlatformStatus string `json:"platform_status"`
	SuccessHint bool   `json:"success_hint"`
	ImagesSet   int    `json:"images_set"`
	TitleFilled bool   `json:"title_filled"`
	Submitted   bool   `json:"submitted"`
	Message     string `json:"message"`
}

// listing1688SellerPublishPayloadOK 严格成功：禁止仅 submitted / 弱 hint 即 published
func listing1688SellerPublishPayloadOK(data listing1688SellerPublishResult) bool {
	platformStatus := strings.ToLower(strings.TrimSpace(data.PlatformStatus))
	return data.OK && data.SuccessHint && data.ImagesSet >= 1 && data.TitleFilled && data.Submitted &&
		strings.TrimSpace(data.OfferID) != "" && (platformStatus == "published" || platformStatus == "auditing")
}

// RunListing1688SellerPublish 向导商家发品：SyncSend Agent，严格校验成功载荷后再写 workflow
func (t *Task) RunListing1688SellerPublish(ctx manager.TaskContext) error {
	var send struct {
		WorkflowID uint            `json:"workflow_id"`
		Title      string          `json:"title"`
		Images     []string        `json:"images"`
		Skus       json.RawMessage `json:"skus"`
		OfferURL   string          `json:"offer_url"`
	}
	if err := json.Unmarshal(ctx.SendJSON, &send); err != nil {
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "解析 Send 失败"))
	}
	if send.WorkflowID == 0 {
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, "workflow_id 缺失"))
	}

	msg := types.AgentMessage[any, any]{
		TaskId:   ctx.TaskId,
		AgentId:  ctx.AgentId,
		Platform: constants.Platform1688,
		Protocol: constants.ProtocolListing1688SellerPublish,
		Send: map[string]any{
			"workflow_id": send.WorkflowID,
			"title":       send.Title,
			"images":      send.Images,
			"skus":        send.Skus,
			"offer_url":   send.OfferURL,
		},
	}
	ctx.Progress.UpdateMessage(ctx.TaskId, "正在商家后台自动发品...")
	raw, err := t.agentManager.SyncSend(ctx.AgentId, msg, 180*time.Second)
	if err != nil {
		t.failListing1688SellerWorkflow(send.WorkflowID, string(constants.Listing1688ErrWizardPublishFail), err.Error())
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, err.Error()))
	}

	var envelope types.AgentMessage[any, json.RawMessage]
	_ = json.Unmarshal(raw, &envelope)
	var resp struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(envelope.Receive, &resp)
	if resp.Code != 0 && resp.Code != 200 {
		code := constants.Listing1688ErrWizardPublishFail
		detail := resp.Msg
		if detail == "" {
			detail = "发品失败"
		}
		if strings.Contains(detail, "SELLER_SESSION_INVALID") {
			code = constants.Listing1688ErrSellerSessionInvalid
		}
		msgText := constants.FormatListing1688WizardError(code, detail)
		t.failListing1688SellerWorkflow(send.WorkflowID, string(code), msgText)
		return fmt.Errorf("%s", msgText)
	}

	var data listing1688SellerPublishResult
	_ = json.Unmarshal(resp.Data, &data)
	if !listing1688SellerPublishPayloadOK(data) {
		detail := data.Message
		if detail == "" {
			detail = "成功判定未通过（需上图+填标题+提交+成功线索）"
		}
		t.failListing1688SellerWorkflow(send.WorkflowID, string(constants.Listing1688ErrWizardPublishFail), detail)
		return fmt.Errorf("%s", constants.FormatListing1688WizardError(constants.Listing1688ErrWizardPublishFail, detail))
	}

	ctx.Progress.UpdateSendOrReceive(ctx.TaskId, nil, resp)
	if t.listing1688WorkflowRepo != nil {
		_ = t.listing1688WorkflowRepo.Update(send.WorkflowID, map[string]interface{}{
			"status":     string(constants.Listing1688StatusPublished),
			"error_code": "",
			"message":    "已提交商家后台发品 offer_id=" + data.OfferID,
		})
	}
	_ = pkg.TimeConverter()
	return nil
}

func (t *Task) failListing1688SellerWorkflow(workflowID uint, code, message string) {
	if t.listing1688WorkflowRepo == nil {
		return
	}
	_ = t.listing1688WorkflowRepo.Update(workflowID, map[string]interface{}{
		"status":     string(constants.Listing1688StatusImgReady),
		"error_code": code,
		"message":    message,
	})
}
