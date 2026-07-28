package utils

import (
	"os"
	"testing"
)

func TestListing1688CookieRoundTrip(t *testing.T) {
	prev := os.Getenv("LISTING_1688_COOKIE_KEY")
	t.Cleanup(func() { _ = os.Setenv("LISTING_1688_COOKIE_KEY", prev) })
	_ = os.Setenv("LISTING_1688_COOKIE_KEY", Listing1688DevCookieKeyBase64())

	enc, err := Listing1688EncryptCookie("a=1; b=2")
	if err != nil {
		t.Fatal(err)
	}
	if enc == "" || enc == "a=1; b=2" {
		t.Fatalf("expected ciphertext, got %q", enc)
	}
	plain, err := Listing1688DecryptCookie(enc)
	if err != nil {
		t.Fatal(err)
	}
	if plain != "a=1; b=2" {
		t.Fatalf("got %q", plain)
	}
}

func TestListing1688CookieKeyMissing(t *testing.T) {
	prev := os.Getenv("LISTING_1688_COOKIE_KEY")
	t.Cleanup(func() { _ = os.Setenv("LISTING_1688_COOKIE_KEY", prev) })
	_ = os.Unsetenv("LISTING_1688_COOKIE_KEY")
	if _, err := Listing1688EncryptCookie("x"); err == nil {
		t.Fatal("expected error when key missing")
	}
}
