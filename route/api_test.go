package route

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
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

func TestBuildStreamEventRowsUsesTopLevelFieldsAsHeaders(t *testing.T) {
	events := []redis.XMessage{
		{
			ID: "1747041200000-0",
			Values: map[string]any{
				"source": "manual",
				"status": "created",
				"data":   `{"do":"test"}`,
			},
		},
	}

	res := buildStreamEventsResponse("orders", events, nil, nil)

	if len(res.Headers) != 3 {
		t.Fatalf("headers = %#v, want 3 top-level fields", res.Headers)
	}
	wantHeaders := []string{"source", "status", "data"}
	for i, want := range wantHeaders {
		if res.Headers[i] != want {
			t.Fatalf("header[%d] = %q, want %q", i, res.Headers[i], want)
		}
	}
	if got := res.Events[0].Values["data"]; got != "{\n  \"do\": \"test\"\n}" {
		t.Fatalf("data value = %q, want pretty object JSON string", got)
	}
	if res.Events[0].Ack.Label != "Pending" || res.Events[0].Ack.State != "pending" {
		t.Fatalf("ack = %#v, want pending ack status", res.Events[0].Ack)
	}
}

func TestBuildStreamEventRowsSummarizesAckedGroups(t *testing.T) {
	events := []redis.XMessage{{ID: "1747041200000-0", Values: map[string]any{"source": "manual"}}}
	acks := map[string]string{
		`1747041200000-0:billing`: `{"group":"billing","consumer":"worker-a","timestamp":"2026-05-12 09:00:00 +08:00 CST"}`,
	}

	res := buildStreamEventsResponse("orders", events, acks, []string{"billing", "email"})

	if got := res.Events[0].Ack.Label; got != "Acked 1/2" {
		t.Fatalf("ack label = %q, want Acked 1/2", got)
	}
	if got := res.Events[0].Ack.State; got != "partial" {
		t.Fatalf("ack state = %q, want partial", got)
	}
	if tooltip := res.Events[0].Ack.Tooltip; !strings.Contains(tooltip, "billing: worker-a") || !strings.Contains(tooltip, "email") {
		t.Fatalf("tooltip = %q, want acked and pending groups", tooltip)
	}
}
