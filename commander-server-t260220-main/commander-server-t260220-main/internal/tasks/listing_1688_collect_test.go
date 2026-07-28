package tasks

import (
	"commander-server-t260220/internal/constants"
	"testing"
)

func TestParse1688NumIid_FromOfferURL(t *testing.T) {
	id, err := Parse1688NumIid("https://detail.1688.com/offer/1234567890.html")
	if err != nil || id != "1234567890" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestParse1688NumIid_PureDigits(t *testing.T) {
	id, err := Parse1688NumIid("9876543210")
	if err != nil || id != "9876543210" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestParse1688NumIid_Invalid(t *testing.T) {
	if _, err := Parse1688NumIid("https://example.com/x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTruncateListing1688Images_Max9Extra(t *testing.T) {
	in := make([]string, 20)
	for i := range in {
		in[i] = "http://x/" + string(rune('a'+i%26))
	}
	out := TruncateListing1688Images(in)
	wantMax := 1 + constants.Listing1688MaxExtraImages
	if len(out) > wantMax {
		t.Fatalf("len=%d want<=%d", len(out), wantMax)
	}
	if out[0] != in[0] {
		t.Fatalf("main image changed: %q", out[0])
	}
}

func TestTruncateListing1688Images_Dedup(t *testing.T) {
	out := TruncateListing1688Images([]string{"http://a/1", "http://a/1", "http://a/2"})
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
}

func TestShouldRunListing1688PlatformAndPeer(t *testing.T) {
	if !shouldRunListing1688PlatformAndPeer(constants.Platform1688, `{"peerUrl":"https://detail.1688.com/offer/1.html"}`) {
		t.Fatal("expected true for 1688+peerUrl")
	}
	if shouldRunListing1688PlatformAndPeer(constants.PlatformTemu, `{"peerUrl":"x"}`) {
		t.Fatal("temu must stay on RunProductIssue")
	}
	if shouldRunListing1688PlatformAndPeer(constants.PlatformAliExpress, `{"peerUrl":"x"}`) {
		t.Fatal("aliexpress must NOT use 1688 path")
	}
	if shouldRunListing1688PlatformAndPeer(constants.Platform1688, `{}`) {
		t.Fatal("1688 without peerUrl must not use listing path")
	}
}
