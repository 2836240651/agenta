package constants

import "strings"

// Listing1688WorkflowStatus 向导主状态机（C2 / 功能规格）
type Listing1688WorkflowStatus string

const (
	Listing1688StatusDraft          Listing1688WorkflowStatus = "draft"
	Listing1688StatusSessionPending Listing1688WorkflowStatus = "session_pending"
	Listing1688StatusSessionValid   Listing1688WorkflowStatus = "session_valid"
	Listing1688StatusCollecting     Listing1688WorkflowStatus = "collecting"
	Listing1688StatusCollected      Listing1688WorkflowStatus = "collected"
	Listing1688StatusImgEditing     Listing1688WorkflowStatus = "img_editing"
	Listing1688StatusImgReady       Listing1688WorkflowStatus = "img_ready"
	Listing1688StatusPublishing     Listing1688WorkflowStatus = "publishing"
	Listing1688StatusPublished      Listing1688WorkflowStatus = "published"
	Listing1688StatusFailed         Listing1688WorkflowStatus = "failed"
)

// Listing1688SessionType 采集会话 vs 商家发品会话（C7）
const (
	Listing1688SessionCollect = "collect"
	Listing1688SessionSeller  = "seller"
)

// 向导专用错误码（与 v1 listing_1688_error 并存；M0 stub 用 NotImplemented）
// AC-05 catalog：Forbidden msg 须以枚举码开头（CODE: 中文），供 Web mapWizardError 解析。
const (
	Listing1688ErrNotImplemented Listing1688ErrorCode = "LISTING_1688_NOT_IMPLEMENTED"
	Listing1688ErrWorkflowNotFound Listing1688ErrorCode = "LISTING_1688_WORKFLOW_NOT_FOUND"
	Listing1688ErrInvalidAgent     Listing1688ErrorCode = "LISTING_1688_INVALID_AGENT"
	// AGENT_OFFLINE：向导专用短码（勿与 v1 LISTING_1688_AGENT_OFFLINE 混用）
	Listing1688ErrWizardAgentOffline Listing1688ErrorCode = "AGENT_OFFLINE"
	Listing1688ErrSessionInvalid       Listing1688ErrorCode = "SESSION_INVALID"
	Listing1688ErrCollectNeedLogin     Listing1688ErrorCode = "COLLECT_NEED_LOGIN"
	Listing1688ErrCollectCaptcha       Listing1688ErrorCode = "COLLECT_CAPTCHA"
	Listing1688ErrCollectParseFail     Listing1688ErrorCode = "COLLECT_PARSE_FAIL"
	Listing1688ErrImagesNotReady       Listing1688ErrorCode = "IMAGES_NOT_READY"
	Listing1688ErrSellerSessionInvalid Listing1688ErrorCode = "SELLER_SESSION_INVALID"
	// 向导发品失败（Web 映射 PUBLISH_FAILED；与 v1 LISTING_1688_PUBLISH_FAILED 并存）
	Listing1688ErrWizardPublishFail Listing1688ErrorCode = "PUBLISH_FAILED"
)

// Listing1688WizardErrorCatalog AC-05 错误文案矩阵（码 → 中文基线，供 Format / Web i18n 对齐）
var Listing1688WizardErrorCatalog = map[Listing1688ErrorCode]string{
	Listing1688ErrWizardAgentOffline:   "Agent 未在线",
	Listing1688ErrSessionInvalid:        "采集会话无效，请重新登录",
	Listing1688ErrCollectNeedLogin:      "请先完成 1688 采集登录",
	Listing1688ErrCollectCaptcha:        "遇到验证码，请在 Agent 浏览器处理后重试",
	Listing1688ErrCollectParseFail:      "竞品解析失败",
	Listing1688ErrImagesNotReady:        "请完成全部图片选定后再发品",
	Listing1688ErrSellerSessionInvalid:  "请先登录 1688 商家中心",
	Listing1688ErrWizardPublishFail:     "商家发品失败，可重试",
	Listing1688ErrWorkflowNotFound:      "工作流不存在或无权访问",
	Listing1688ErrInvalidAgent:          "Agent 无效",
}

// listing1688WizardErrorParseOrder 长码优先，避免前缀误匹配
var listing1688WizardErrorParseOrder = []Listing1688ErrorCode{
	Listing1688ErrWorkflowNotFound,
	Listing1688ErrInvalidAgent,
	Listing1688ErrSellerSessionInvalid,
	Listing1688ErrCollectNeedLogin,
	Listing1688ErrCollectParseFail,
	Listing1688ErrCollectCaptcha,
	Listing1688ErrImagesNotReady,
	Listing1688ErrSessionInvalid,
	Listing1688ErrWizardPublishFail,
	Listing1688ErrWizardAgentOffline,
}

// FormatListing1688WizardError 生成 `CODE: 中文`；detail 非空时用 detail 替换基线正文（仍带码前缀）。
func FormatListing1688WizardError(code Listing1688ErrorCode, detail string) string {
	detail = strings.TrimSpace(detail)
	if detail != "" {
		if detail == string(code) || strings.HasPrefix(detail, string(code)+":") {
			if detail == string(code) {
				if base, ok := Listing1688WizardErrorCatalog[code]; ok && base != "" {
					return string(code) + ": " + base
				}
				return string(code)
			}
			return detail
		}
		return string(code) + ": " + detail
	}
	if base, ok := Listing1688WizardErrorCatalog[code]; ok && base != "" {
		return string(code) + ": " + base
	}
	return string(code)
}

// ParseListing1688WizardErrorCode 从 Forbidden msg / error_code 解析 AC-05 catalog 码；未知返回空。
func ParseListing1688WizardErrorCode(msg string) Listing1688ErrorCode {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	for _, code := range listing1688WizardErrorParseOrder {
		s := string(code)
		if msg == s || strings.HasPrefix(msg, s+":") || strings.HasPrefix(msg, s+" ") {
			return code
		}
	}
	return ""
}

// Listing1688WizardDefaultPrompt Step3 默认图优提示（功能规格）
const Listing1688WizardDefaultPrompt = "保持商品主体，优化背景光影与构图，适合电商主图。"

// listing_1688_image_jobs.status（M2）
const (
	Listing1688ImagePending     = "pending"
	Listing1688ImageRunning     = "running"
	Listing1688ImageDone        = "done"
	Listing1688ImageFailed      = "failed"
	Listing1688ImageUseOriginal = "use_original"
	Listing1688ImageSelected    = "selected"
)

// Listing1688MaxImageSlots = 主图 1 + 副图上限
const Listing1688MaxImageSlots = 1 + Listing1688MaxExtraImages
