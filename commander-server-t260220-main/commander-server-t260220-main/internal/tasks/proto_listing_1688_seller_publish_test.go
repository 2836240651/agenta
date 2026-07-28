package tasks

import "testing"

func TestListing1688SellerPublishPayloadOK_RejectsWeakSuccess(t *testing.T) {
	cases := []struct {
		name string
		data listing1688SellerPublishResult
		want bool
	}{
		{
			name: "only submitted",
			data: listing1688SellerPublishResult{OK: true, Submitted: true},
			want: false,
		},
		{
			name: "submitted without images",
			data: listing1688SellerPublishResult{OK: true, Submitted: true, TitleFilled: true, SuccessHint: true, ImagesSet: 0},
			want: false,
		},
		{
			name: "no success_hint",
			data: listing1688SellerPublishResult{OK: true, Submitted: true, TitleFilled: true, ImagesSet: 2, SuccessHint: false},
			want: false,
		},
		{
			name: "missing offer evidence",
			data: listing1688SellerPublishResult{OK: true, Submitted: true, TitleFilled: true, ImagesSet: 1, SuccessHint: true},
			want: false,
		},
		{
			name: "auditing offer",
			data: listing1688SellerPublishResult{OK: true, OfferID: "123456789", PlatformStatus: "auditing", Submitted: true, TitleFilled: true, ImagesSet: 1, SuccessHint: true},
			want: true,
		},
	}
	for _, tc := range cases {
		if got := listing1688SellerPublishPayloadOK(tc.data); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
