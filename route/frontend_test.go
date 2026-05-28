package route

import (
	"encoding/json"
	"testing"
)

func TestStreamListItemSerializesForReactHome(t *testing.T) {
	payload, err := json.Marshal(StreamListItem{
		Name:   "stream:orders",
		Length: 2,
		Groups: 1,
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}

	got := string(payload)
	want := `{"name":"stream:orders","displayName":"","length":2,"groups":1}`
	if got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}
}
