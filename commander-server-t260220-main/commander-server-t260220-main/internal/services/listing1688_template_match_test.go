package services

import (
	"commander-server-t260220/internal/modules"
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestListing1688RankTemplateCandidates(t *testing.T) {
	workflow := &modules.TableListing1688Workflows{
		Model: gorm.Model{ID: 1},
		OfferSnapshot: datatypes.JSON([]byte(`{
			"title":"高碳钢章鱼钩带倒刺鱼钩",
			"price_min":0.08,
			"price_max":0.12,
			"skus":[{"name":"A"},{"name":"B"}]
		}`)),
	}
	products := []listing1688TemplateProduct{
		{OfferID: "1", Title: "高碳钢章鱼钩带倒刺回头钩", PriceMin: floatPtr(0.08), PriceMax: floatPtr(0.1)},
		{OfferID: "2", Title: "浮水珠珠泡沫浮球", PriceMin: floatPtr(0.2), PriceMax: floatPtr(0.21)},
	}
	got := listing1688RankTemplateCandidates(workflow, products, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 candidates, got %d", len(got))
	}
	if got[0].OfferID != "1" {
		t.Fatalf("best candidate=%s", got[0].OfferID)
	}
	if got[0].Score <= got[1].Score {
		t.Fatalf("scores not ordered: %+v", got)
	}
}

func TestListing1688WorkflowTemplateMatchResult(t *testing.T) {
	candidates := []listing1688TemplateCandidate{{OfferID: "1001", Title: "测试母版", Score: 18}}
	raw, _ := json.Marshal(candidates)
	row := &modules.TableListing1688Workflows{
		MatchStatus:            listing1688MatchStatusSelected,
		MatchedTemplateOfferID: "1001",
		MatchedCategoryID:      "1044663",
		SimilarPageURL:         "https://offer-new.1688.com/popular/publish.htm?catId=1044663",
		MatchCandidates:        datatypes.JSON(raw),
	}
	result := listing1688WorkflowTemplateMatchResult(row)
	if result["selected_offer_id"] != "1001" {
		t.Fatalf("selected_offer_id=%v", result["selected_offer_id"])
	}
	list, ok := result["candidates"].([]listing1688TemplateCandidate)
	if !ok || len(list) != 1 {
		t.Fatalf("candidates=%#v", result["candidates"])
	}
}

func floatPtr(v float64) *float64 { return &v }
