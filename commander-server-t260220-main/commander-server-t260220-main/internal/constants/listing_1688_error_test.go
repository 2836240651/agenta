package constants

import "testing"

func TestListing1688MaxExtraImages(t *testing.T) {
	if Listing1688MaxExtraImages != 9 {
		t.Fatalf("want 9, got %d", Listing1688MaxExtraImages)
	}
}

func TestListing1688ErrCodesStable(t *testing.T) {
	if Listing1688ErrCollectNoImage != "LISTING_1688_COLLECT_NO_IMAGE" {
		t.Fatal(Listing1688ErrCollectNoImage)
	}
}
