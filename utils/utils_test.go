package utils

import "testing"

func TestBase64EncodeReturnsURLSafeEncodingWithoutPadding(t *testing.T) {
	got := Base64Encode("orders:created/paid+refunded")
	want := "b3JkZXJzOmNyZWF0ZWQvcGFpZCtyZWZ1bmRlZA"

	if got != want {
		t.Fatalf("Base64Encode() = %q, want %q", got, want)
	}
}
