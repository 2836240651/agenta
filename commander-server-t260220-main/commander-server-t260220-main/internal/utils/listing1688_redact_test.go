package utils

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestListing1688RedactCookiePlain(t *testing.T) {
	if got := Listing1688RedactCookiePlain(""); got != "" {
		t.Fatalf("empty -> %q", got)
	}
	got := Listing1688RedactCookiePlain("a=1; b=2")
	if strings.Contains(got, "a=1") {
		t.Fatalf("leaked plaintext: %q", got)
	}
	if !strings.Contains(got, "len=8") {
		t.Fatalf("want length hint, got %q", got)
	}
}

func TestListing1688RedactCookieInJSON_Nested(t *testing.T) {
	raw := []byte(`{"protocol":"listing1688_export_cookie","receive":{"code":0,"data":{"cookie":"secret=abc; x=1"}},"send":{"cookie":"probe=val"}}`)
	out := Listing1688RedactCookieInJSON(raw)
	if strings.Contains(out, "secret=abc") || strings.Contains(out, "probe=val") {
		t.Fatalf("leaked cookie: %s", out)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("out not json: %v / %s", err, out)
	}
	recv, _ := v["receive"].(map[string]any)
	data, _ := recv["data"].(map[string]any)
	if s, _ := data["cookie"].(string); !strings.HasPrefix(s, "***") {
		t.Fatalf("receive.data.cookie=%q", s)
	}
	send, _ := v["send"].(map[string]any)
	if s, _ := send["cookie"].(string); !strings.HasPrefix(s, "***") {
		t.Fatalf("send.cookie=%q", s)
	}
}

func TestListing1688RedactCookieInJSON_InvalidJSON(t *testing.T) {
	out := Listing1688RedactCookieInJSON([]byte(`{"cookie":"plain-leak"`))
	if strings.Contains(out, "plain-leak") {
		t.Fatalf("fallback leaked: %s", out)
	}
}

func TestListing1688RedactCookieInJSON_CaseInsensitiveKey(t *testing.T) {
	raw := []byte(`{"Cookie":"SECRET=1","nested":{"COOKIE":"SECRET=2"}}`)
	out := Listing1688RedactCookieInJSON(raw)
	if strings.Contains(out, "SECRET=") {
		t.Fatalf("case-insensitive key leaked: %s", out)
	}
}
