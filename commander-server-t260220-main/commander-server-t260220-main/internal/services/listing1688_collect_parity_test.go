package services

import (
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

// TC-C-05: Server M2/M3 consumers lock on wizard canonical snapshot keys.
func TestListing1688SnapshotCanonicalFields_M2M3(t *testing.T) {
	snap := map[string]any{
		"ok":             true,
		"status":         "success",
		"url":            "https://detail.1688.com/offer/9876543210123.html",
		"source_item_id": "9876543210123",
		"title":          "TC-C-05 对齐样例标题",
		"main_images": []string{
			"https://cbu01.alicdn.com/img/ibank/fixture_main_1.jpg",
			"https://cbu01.alicdn.com/img/ibank/fixture_main_2.jpg",
		},
		"skus": []map[string]any{
			{
				"sku_key":       "红色>M",
				"spec_attrs":    "红色>M",
				"name":          "红色>M",
				"price":         18.5,
				"source_sku_id": "sku-red-m",
				"spec_id":       "spec-red-m",
			},
		},
		"price_min": 18.5,
		"price_max": 20.0,
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}

	urls, err := listing1688ParseSnapshotMainImages(datatypes.JSON(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) < 1 {
		t.Fatal("M2 main_images empty")
	}

	skus := listing1688SnapshotSkus(datatypes.JSON(raw))
	if len(skus) < 1 {
		t.Fatal("M3 skus empty")
	}
	sku0 := skus[0]
	if sku0["price"] == nil {
		t.Fatal("sku price required for M3")
	}
	if sku0["spec_attrs"] == nil && sku0["name"] == nil {
		t.Fatal("sku needs spec_attrs or name for M3")
	}

	var decoded listing1688OfferSnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Title == "" || decoded.OfferID == "" || len(decoded.MainImages) < 1 {
		t.Fatalf("collect struct: %+v", decoded)
	}
	if decoded.PriceMin == nil || decoded.PriceMax == nil {
		t.Fatal("price_min/price_max required when source has prices")
	}
	if decoded.URL == "" {
		t.Fatal("url required for traceability")
	}
}

func TestListing1688ParseSnapshotMainImages_RejectsImagesAliasOnly(t *testing.T) {
	// Non-canonical root key `images` must NOT satisfy M2 (canonical is main_images).
	raw, _ := json.Marshal(map[string]any{
		"title":  "t",
		"images": []string{"https://cbu01.alicdn.com/a.jpg"},
	})
	_, err := listing1688ParseSnapshotMainImages(datatypes.JSON(raw))
	if err == nil {
		t.Fatal("expected error when only images alias present")
	}
}
