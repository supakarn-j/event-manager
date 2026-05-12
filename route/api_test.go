package route

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestParseEventPayloadAcceptsJSONObject(t *testing.T) {
	form := url.Values{}
	form.Set("payload", `{"order_id":189,"status":"created","manual":true,"tags":["ops","test"]}`)

	values, err := parseEventPayload(strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("parseEventPayload returned error: %v", err)
	}

	if values["order_id"] != "189" {
		t.Fatalf("order_id = %q, want %q", values["order_id"], "189")
	}
	if values["status"] != "created" {
		t.Fatalf("status = %q, want %q", values["status"], "created")
	}
	if values["manual"] != "true" {
		t.Fatalf("manual = %q, want %q", values["manual"], "true")
	}
	if values["tags"] != `["ops","test"]` {
		t.Fatalf("tags = %q, want JSON array string", values["tags"])
	}
}

func TestParseEventPayloadRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{
			name:        "empty form payload",
			contentType: "application/x-www-form-urlencoded",
			body:        url.Values{"payload": {""}}.Encode(),
		},
		{
			name:        "array json",
			contentType: "application/json",
			body:        `["not","an","object"]`,
		},
		{
			name:        "empty object",
			contentType: "application/json",
			body:        `{}`,
		},
		{
			name:        "invalid json",
			contentType: "application/json",
			body:        `{nope`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseEventPayload(strings.NewReader(tt.body), tt.contentType); err == nil {
				t.Fatal("parseEventPayload returned nil error, want error")
			}
		})
	}
}

func TestParseEventPayloadAcceptsRawJSONRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/api/v1/streams/orders/events", strings.NewReader(`{"source":"manual"}`))
	if err != nil {
		t.Fatalf("NewRequest returned error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	values, err := parseEventPayload(req.Body, req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parseEventPayload returned error: %v", err)
	}

	if values["source"] != "manual" {
		t.Fatalf("source = %q, want %q", values["source"], "manual")
	}
}

func TestAckFieldPatternForEventMatchesGroupedAckFields(t *testing.T) {
	if got := ackFieldPatternForEvent("1747041200000-0"); got != "1747041200000-0:*" {
		t.Fatalf("pattern = %q, want grouped ack field pattern", got)
	}
}
