package services

import (
	"reflect"
	"testing"
)

func TestListing1688ResolvedTemplateRequiredFields_PrefersObservedSellerPage(t *testing.T) {
	existing := []string{"title", "legacy_field"}
	observed := []string{"产品类别", "品牌", "图文详情"}
	want := []string{"产品类别", "品牌", "图文详情"}
	if got := listing1688ResolvedTemplateRequiredFields(existing, observed); !reflect.DeepEqual(got, want) {
		t.Fatalf("required fields=%#v, want=%#v", got, want)
	}
}

func TestListing1688ResolvedTemplateRequiredFields_RetainsExistingWhenAgentReturnsNothing(t *testing.T) {
	existing := []string{"title", "brand"}
	if got := listing1688ResolvedTemplateRequiredFields(existing, nil); !reflect.DeepEqual(got, existing) {
		t.Fatalf("required fields=%#v, want=%#v", got, existing)
	}
}