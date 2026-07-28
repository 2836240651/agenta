package constants

import (
	"strings"
	"testing"
)

// AC-05 Error catalog — 每个码至少断言 Format 结果含码与基线中文
func TestListing1688WizardErrorCatalog_FormatContainsCode(t *testing.T) {
	want := []struct {
		code Listing1688ErrorCode
		zh   string
	}{
		{Listing1688ErrWizardAgentOffline, "Agent 未在线"},
		{Listing1688ErrSessionInvalid, "采集会话无效，请重新登录"},
		{Listing1688ErrCollectNeedLogin, "请先完成 1688 采集登录"},
		{Listing1688ErrCollectCaptcha, "遇到验证码，请在 Agent 浏览器处理后重试"},
		{Listing1688ErrCollectParseFail, "竞品解析失败"},
		{Listing1688ErrImagesNotReady, "请完成全部图片选定后再发品"},
		{Listing1688ErrSellerSessionInvalid, "请先登录 1688 商家中心"},
		{Listing1688ErrWizardPublishFail, "商家发品失败，可重试"},
		{Listing1688ErrWorkflowNotFound, "工作流不存在或无权访问"},
		{Listing1688ErrInvalidAgent, "Agent 无效"},
	}
	if len(want) != len(Listing1688WizardErrorCatalog) {
		t.Fatalf("catalog size mismatch: cases=%d map=%d", len(want), len(Listing1688WizardErrorCatalog))
	}
	for _, tc := range want {
		t.Run(string(tc.code), func(t *testing.T) {
			gotBase, ok := Listing1688WizardErrorCatalog[tc.code]
			if !ok {
				t.Fatalf("missing catalog entry for %s", tc.code)
			}
			if gotBase != tc.zh {
				t.Fatalf("zh baseline: want %q got %q", tc.zh, gotBase)
			}
			msg := FormatListing1688WizardError(tc.code, "")
			if !strings.Contains(msg, string(tc.code)) {
				t.Fatalf("msg must contain code: %q", msg)
			}
			if !strings.HasPrefix(msg, string(tc.code)+":") {
				t.Fatalf("msg must start with CODE:: %q", msg)
			}
			if !strings.Contains(msg, tc.zh) {
				t.Fatalf("msg must contain baseline zh: %q", msg)
			}
			withDetail := FormatListing1688WizardError(tc.code, "补充细节")
			if !strings.Contains(withDetail, string(tc.code)) {
				t.Fatalf("detail msg must contain code: %q", withDetail)
			}
			if !strings.Contains(withDetail, "补充细节") {
				t.Fatalf("detail msg must contain detail: %q", withDetail)
			}
		})
	}
}

func TestListing1688ErrWizardAgentOffline_IsAGENT_OFFLINE(t *testing.T) {
	if Listing1688ErrWizardAgentOffline != "AGENT_OFFLINE" {
		t.Fatalf("want AGENT_OFFLINE, got %s", Listing1688ErrWizardAgentOffline)
	}
	// v1 一键上架码不得被向导覆盖
	if Listing1688ErrAgentOffline != "LISTING_1688_AGENT_OFFLINE" {
		t.Fatalf("v1 agent offline drifted: %s", Listing1688ErrAgentOffline)
	}
}

func TestParseListing1688WizardErrorCode(t *testing.T) {
	cases := []struct {
		msg  string
		want Listing1688ErrorCode
	}{
		{"AGENT_OFFLINE: Agent 未在线", Listing1688ErrWizardAgentOffline},
		{"IMAGES_NOT_READY: 请完成全部图片选定后再发品", Listing1688ErrImagesNotReady},
		{"SELLER_SESSION_INVALID: 请先登录 1688 商家中心", Listing1688ErrSellerSessionInvalid},
		{"LISTING_1688_WORKFLOW_NOT_FOUND: 工作流不存在或无权访问", Listing1688ErrWorkflowNotFound},
		{"unknown noise", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := ParseListing1688WizardErrorCode(tc.msg)
		if got != tc.want {
			t.Fatalf("msg=%q want %q got %q", tc.msg, tc.want, got)
		}
	}
}
