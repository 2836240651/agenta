package constants

import "testing"

func TestListing1688WizardStatusDraft(t *testing.T) {
	if Listing1688StatusDraft != "draft" {
		t.Fatalf("want draft, got %s", Listing1688StatusDraft)
	}
}

func TestListing1688ErrNotImplementedStable(t *testing.T) {
	if Listing1688ErrNotImplemented != "LISTING_1688_NOT_IMPLEMENTED" {
		t.Fatal(Listing1688ErrNotImplemented)
	}
}

func TestListing1688SessionTypes(t *testing.T) {
	if Listing1688SessionCollect != "collect" || Listing1688SessionSeller != "seller" {
		t.Fatal("session type constants drifted")
	}
}
