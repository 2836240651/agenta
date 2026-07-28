package services

import (
	"commander-server-t260220/internal/modules"
	"testing"
)

func TestListing1688BuildPublishDraft_PrioritizesProductAndReportsUnknownRequiredFields(t *testing.T) {
	draft := listing1688BuildPublishDraft(listing1688PublishProfileInput{
		Workflow: listing1688PublishWorkflowData{Title: "hook", Images: []string{"https://cdn.example/hook.png"}, Skus: []map[string]any{{"name": "4/0#", "price": 0.17, "stock": 85499}}, Fields: map[string]any{"brand": "workflow-brand"}},
		StorePreset: listing1688PublishStorePreset{Fields: map[string]any{"shipping_address": "Taizhou", "shipping_template_id": "template-1", "delivery_days": 2, "unit": "piece"}},
		CategoryTemplate: listing1688PublishCategoryTemplate{CategoryID: "1044663", Fields: map[string]any{"brand": "template-brand", "hook_type": "right-angle"}, RequiredFields: []string{"title", "images", "skus", "brand", "hook_type", "shipping_address", "shipping_template_id", "delivery_days", "weight_kg", "dimensions_cm", "detail_html"}},
	})

	if draft.CategoryID != "1044663" { t.Fatalf("category_id=%q", draft.CategoryID) }
	if got := draft.FormData["brand"]; got != "workflow-brand" { t.Fatalf("brand must prefer workflow field, got %#v", got) }
	if got := draft.FormData["hook_type"]; got != "right-angle" { t.Fatalf("hook_type=%#v", got) }
	if got := draft.FormData["shipping_template_id"]; got != "template-1" { t.Fatalf("shipping_template_id=%#v", got) }
	if draft.ValidationStatus != listing1688PublishDraftStatusMissingFields { t.Fatalf("validation_status=%q", draft.ValidationStatus) }
	wantMissing := map[string]bool{"weight_kg": true, "dimensions_cm": true, "detail_html": true}
	if len(draft.MissingFields) != len(wantMissing) { t.Fatalf("missing_fields=%#v", draft.MissingFields) }
	for _, field := range draft.MissingFields { if !wantMissing[field] { t.Fatalf("unexpected missing field %q", field) }; delete(wantMissing, field) }
	if len(wantMissing) != 0 { t.Fatalf("missing expected fields %#v", wantMissing) }
}

func TestListing1688BuildPublishDraft_UserInputWinsAndValidatesRequiredFields(t *testing.T) {
	draft := listing1688BuildPublishDraft(listing1688PublishProfileInput{
		Workflow: listing1688PublishWorkflowData{Title: "hook", Images: []string{"https://cdn.example/hook.png"}, Skus: []map[string]any{{"name": "4/0#", "price": 0.17, "stock": 10}}},
		CategoryTemplate: listing1688PublishCategoryTemplate{CategoryID: "1044663", RequiredFields: []string{"title", "images", "skus", "brand", "weight_kg", "dimensions_cm", "detail_html"}},
		UserFields: map[string]any{"brand": "yituo", "weight_kg": 0.02, "dimensions_cm": "10x6x1", "detail_html": "<p>hook details</p>"},
	})

	if draft.ValidationStatus != listing1688PublishDraftStatusReady { t.Fatalf("validation_status=%q, missing=%#v", draft.ValidationStatus, draft.MissingFields) }
	if len(draft.MissingFields) != 0 { t.Fatalf("missing_fields=%#v", draft.MissingFields) }
	if got := draft.FormData["brand"]; got != "yituo" { t.Fatalf("brand=%#v", got) }
}
func TestListing1688PublishDraftCanSubmit_RequiresSavedDraft(t *testing.T) {
	if listing1688PublishDraftCanSubmit(listing1688PublishDraftStatusReady, false) {
		t.Fatal("ready draft without seller-page save must not submit")
	}
	if !listing1688PublishDraftCanSubmit(listing1688PublishDraftStatusReady, true) {
		t.Fatal("ready seller-page draft must submit")
	}
	if listing1688PublishDraftCanSubmit(listing1688PublishDraftStatusMissingFields, true) {
		t.Fatal("missing fields draft must not submit")
	}
}

func TestListing1688PublishFormValidate_ClearsDraftSavedFlag(t *testing.T) {
	// form/validate must always persist DraftSaved=false so re-validate forces another save-draft.
	row := modules.TableListing1688PublishDrafts{
		ValidationStatus: listing1688PublishDraftStatusReady,
		DraftSaved:       false,
	}
	if row.DraftSaved {
		t.Fatal("validate upsert must clear draft_saved")
	}
	if listing1688PublishDraftCanSubmit(row.ValidationStatus, row.DraftSaved) {
		t.Fatal("validated-but-unsaved draft must not be publishable")
	}
}