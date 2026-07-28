package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Listing1688RedactCookiePlain 日志用：永不输出 Cookie 明文，仅长度。
func Listing1688RedactCookiePlain(cookie string) string {
	if cookie == "" {
		return ""
	}
	return fmt.Sprintf("*** (len=%d)", len(cookie))
}

// Listing1688RedactCookieInJSON 递归脱敏 JSON 中名为 cookie 的字符串字段（大小写不敏感）。
// 解析失败时回退为字面量替换，仍避免原样打印。
func Listing1688RedactCookieInJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return listing1688RedactCookieFallback(string(raw))
	}
	listing1688RedactCookieValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return "***"
	}
	return string(out)
}

func listing1688RedactCookieValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if strings.EqualFold(k, "cookie") {
				if s, ok := val.(string); ok {
					t[k] = Listing1688RedactCookiePlain(s)
					continue
				}
			}
			listing1688RedactCookieValue(val)
		}
	case []any:
		for _, item := range t {
			listing1688RedactCookieValue(item)
		}
	}
}

func listing1688RedactCookieFallback(s string) string {
	// 粗粒度兜底：含 "cookie" 键痕迹时整段不原样落日志
	lower := strings.ToLower(s)
	if strings.Contains(lower, `"cookie"`) || strings.Contains(lower, `"cookie":`) {
		return fmt.Sprintf("*** (unparsed json, len=%d, cookie_key_present)", len(s))
	}
	return s
}
