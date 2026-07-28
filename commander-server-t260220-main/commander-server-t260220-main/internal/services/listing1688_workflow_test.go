package services

import (
	"commander-server-t260220/internal/constants"
	"testing"
)

func TestListing1688StubMilestoneCodes(t *testing.T) {
	if constants.Listing1688ErrNotImplemented != "LISTING_1688_NOT_IMPLEMENTED" {
		t.Fatal(constants.Listing1688ErrNotImplemented)
	}
	if constants.Listing1688StatusDraft != "draft" {
		t.Fatal(constants.Listing1688StatusDraft)
	}
}

func TestListing1688ErrWorkflowNotFoundStable(t *testing.T) {
	if constants.Listing1688ErrWorkflowNotFound != "LISTING_1688_WORKFLOW_NOT_FOUND" {
		t.Fatal(constants.Listing1688ErrWorkflowNotFound)
	}
}
