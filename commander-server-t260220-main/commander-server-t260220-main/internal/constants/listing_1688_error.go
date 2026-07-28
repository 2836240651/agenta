package constants

// Listing1688ErrorCode 1688 采集图优上架错误码（稳定契约，供 Web 映射文案）
type Listing1688ErrorCode string

const Listing1688MaxExtraImages = 9

// Listing1688Img2ImgPrompt MVP 固定图生图提示（1688 专用，勿与 Temu 五场景混用）
const Listing1688Img2ImgPrompt = "在保持商品主体、结构与材质真实一致的前提下，优化背景与光影，生成适合 1688 商品主图/详情图的高质量电商图，不要改变商品品类与形状。"

const (
	Listing1688ErrInvalidPeerURL     Listing1688ErrorCode = "LISTING_1688_INVALID_PEER_URL"
	Listing1688ErrCollectFailed      Listing1688ErrorCode = "LISTING_1688_COLLECT_FAILED"
	Listing1688ErrCollectNoImage     Listing1688ErrorCode = "LISTING_1688_COLLECT_NO_IMAGE"
	Listing1688ErrImg2ImgFailed      Listing1688ErrorCode = "LISTING_1688_IMG2IMG_FAILED"
	Listing1688ErrImg2ImgPartialFail Listing1688ErrorCode = "LISTING_1688_IMG2IMG_PARTIAL_FAIL"
	Listing1688ErrAgentOffline       Listing1688ErrorCode = "LISTING_1688_AGENT_OFFLINE"
	Listing1688ErrPublishFailed      Listing1688ErrorCode = "LISTING_1688_PUBLISH_FAILED"
	Listing1688ErrPublishUnsupported Listing1688ErrorCode = "LISTING_1688_PUBLISH_UNSUPPORTED"
	Listing1688ErrInternal           Listing1688ErrorCode = "LISTING_1688_INTERNAL"
)
