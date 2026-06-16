package utils

import "testing"

func TestBase64EncodeReturnsURLSafeEncodingWithoutPadding(t *testing.T) {
	got := Base64Encode("orders:created/paid+refunded")
	want := "b3JkZXJzOmNyZWF0ZWQvcGFpZCtyZWZ1bmRlZA"

	if got != want {
		t.Fatalf("Base64Encode() = %q, want %q", got, want)
	}
}

func TestGetGroupAndConsumerFromPubSubKey(t *testing.T) {
	key := "consumer:health:billing:consumer01"

	if got := GetGroupFromPubSub(key); got != "billing" {
		t.Fatalf("GetGroupFromPubSub() = %q, want billing", got)
	}
	if got := GetConsumerFromPubSub(key); got != "consumer01" {
		t.Fatalf("GetConsumerFromPubSub() = %q, want consumer01", got)
	}
}

func TestGetGroupAndConsumerFromMalformedPubSubKey(t *testing.T) {
	key := "consumer:health"

	if got := GetGroupFromPubSub(key); got != "" {
		t.Fatalf("GetGroupFromPubSub() = %q, want empty", got)
	}
	if got := GetConsumerFromPubSub(key); got != "" {
		t.Fatalf("GetConsumerFromPubSub() = %q, want empty", got)
	}
}
