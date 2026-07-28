package tasks

import (
	"commander-server-t260220/internal/constants"
	"commander-server-t260220/internal/utils"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var re1688OfferID = regexp.MustCompile(`(?:offer/|offerId=|num_iid=)(\d+)`)

// Parse1688NumIid 从同行链接或纯数字 ID 解析 num_iid。
func Parse1688NumIid(peerURL string) (string, error) {
	s := strings.TrimSpace(peerURL)
	if s == "" {
		return "", fmt.Errorf("peer url empty")
	}
	if matched, _ := regexp.MatchString(`^\d+$`, s); matched {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid peer url: %w", err)
	}
	host := strings.ToLower(u.Host)
	if !strings.Contains(host, "1688.com") {
		return "", fmt.Errorf("not a 1688 url")
	}
	if m := re1688OfferID.FindStringSubmatch(s); len(m) >= 2 {
		return m[1], nil
	}
	return "", fmt.Errorf("num_iid not found in url")
}

// TruncateListing1688Images 保留主图 + 副图上限（Listing1688MaxExtraImages）。
func TruncateListing1688Images(urls []string) []string {
	seen := make(map[string]struct{}, len(urls))
	out := make([]string, 0, 1+constants.Listing1688MaxExtraImages)
	for _, raw := range urls {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		if len(out) == 0 {
			out = append(out, u)
			continue
		}
		if len(out)-1 >= constants.Listing1688MaxExtraImages {
			break
		}
		out = append(out, u)
	}
	return out
}

// CollectPeerImages1688 通过 Onebound 拉取同行主图与副图 URL 列表。
func CollectPeerImages1688(onebound *utils.OneboundUtils, numIid string) (title string, urls []string, err error) {
	detail, err := onebound.GetProductDetailBy1688(numIid)
	if err != nil {
		return "", nil, err
	}
	if detail == nil {
		return "", nil, fmt.Errorf("empty detail")
	}
	title = strings.TrimSpace(detail.Item.Title)
	candidates := make([]string, 0, 1+len(detail.Item.ItemImgs))
	if pic := strings.TrimSpace(detail.Item.PicURL); pic != "" {
		candidates = append(candidates, pic)
	}
	for _, img := range detail.Item.ItemImgs {
		if u := strings.TrimSpace(img.URL); u != "" {
			candidates = append(candidates, u)
		}
	}
	urls = TruncateListing1688Images(candidates)
	if len(urls) == 0 {
		return title, nil, fmt.Errorf("%s", constants.Listing1688ErrCollectNoImage)
	}
	return title, urls, nil
}

// CollectPeerImagesFromURL 解析链接后采集图片。
func CollectPeerImagesFromURL(onebound *utils.OneboundUtils, peerURL string) (numIid, title string, urls []string, err error) {
	numIid, err = Parse1688NumIid(peerURL)
	if err != nil {
		return "", "", nil, err
	}
	title, urls, err = CollectPeerImages1688(onebound, numIid)
	if err != nil {
		return numIid, title, urls, err
	}
	return numIid, title, urls, nil
}

// IsListing1688CollectNoImageErr 判断是否无图错误（供上层映射错误码）。
func IsListing1688CollectNoImageErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), string(constants.Listing1688ErrCollectNoImage))
}

type listing1688SendPeek struct {
	PeerURL string `json:"peerUrl"`
}

// ShouldRunListing1688 判断是否走 1688 采集图优旁路（platform=1688 且含 peerUrl）。
func ShouldRunListing1688(platform string, sendJSON []byte) bool {
	if platform != constants.Platform1688 {
		return false
	}
	var peek listing1688SendPeek
	if err := json.Unmarshal(sendJSON, &peek); err != nil {
		return false
	}
	return strings.TrimSpace(peek.PeerURL) != ""
}

func shouldRunListing1688PlatformAndPeer(platform, sendJSON string) bool {
	return ShouldRunListing1688(platform, []byte(sendJSON))
}
