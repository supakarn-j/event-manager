package route

import (
	"bytes"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestCreateMyRendererParsesTemplates(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	createMyRenderer()
}

func TestStreamsTemplateRendersStandaloneStream(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	renderer := createMyRenderer()
	instance := renderer.Instance("streams", map[string]any{
		"tree": buildStreamTree([]StreamListItem{
			{Name: "test", Length: 0, Groups: 0},
		}),
		"redisURL": "redis://localhost:6379",
	})

	recorder := httptest.NewRecorder()
	if err := instance.Render(recorder); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("/streams/test")) {
		t.Fatalf("rendered template does not contain standalone stream link: %s", recorder.Body.String())
	}
}

func TestStreamDetailTemplateRendersGroupPendingCount(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}()

	renderer := createMyRenderer()
	instance := renderer.Instance("streams_name", map[string]any{
		"title": "orders",
		"name":  "orders",
		"groups": []map[string]any{
			{
				"Name":    "workers",
				"Pending": int64(12),
				"Consumers": []map[string]any{
					{
						"Name":     "worker-1",
						"LastSeen": "2026-05-11 12:00:00 +00:00 UTC",
						"Healthy":  true,
						"Pending":  int64(7),
					},
				},
			},
		},
	})

	recorder := httptest.NewRecorder()
	if err := instance.Render(recorder); err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("PEL: 12")) {
		t.Fatalf("rendered template does not contain PEL count: %s", recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("PEL: 7")) {
		t.Fatalf("rendered template does not contain consumer PEL count: %s", recorder.Body.String())
	}
}

func TestBuildStreamTreeGroupsColonSeparatedNames(t *testing.T) {
	streams := []StreamListItem{
		{Name: "b", Length: 3, Groups: 1},
		{Name: "a:yy", Length: 7, Groups: 2},
		{Name: "a:xx", Length: 5, Groups: 1},
		{Name: "z:deep:item", Length: 11, Groups: 4},
		{Name: "stream:test:123", Length: 13, Groups: 5},
	}

	tree := buildStreamTree(streams)

	if len(tree.Nodes) != 4 {
		t.Fatalf("root node count = %d, want 4", len(tree.Nodes))
	}
	if tree.Nodes[0].Name != "a" {
		t.Fatalf("first root = %q, want %q", tree.Nodes[0].Name, "a")
	}
	if got := tree.Nodes[0].Children[0].DisplayName; got != "xx" {
		t.Fatalf("first child display = %q, want %q", got, "xx")
	}
	if got := tree.Nodes[0].Children[1].DisplayName; got != "yy" {
		t.Fatalf("second child display = %q, want %q", got, "yy")
	}
	if got := tree.Nodes[1].Children[0].Children[0].DisplayName; got != "123" {
		t.Fatalf("nested child display = %q, want %q", got, "123")
	}
	if got := tree.Nodes[2].Children[0].Children[0].DisplayName; got != "item" {
		t.Fatalf("deep z child display = %q, want %q", got, "item")
	}
	if !tree.Nodes[3].IsStream || tree.Nodes[3].Name != "b" {
		t.Fatalf("fourth root = %+v, want standalone stream b", tree.Nodes[3])
	}
	if tree.TotalStreams != 5 {
		t.Fatalf("total streams = %d, want 5", tree.TotalStreams)
	}
}

func TestBuildEventTableUsesTopLevelValueKeysAsColumns(t *testing.T) {
	table := buildEventTable([]redis.XMessage{
		{
			ID: "1-0",
			Values: map[string]any{
				"source": "manual",
				"status": "created",
				"data": map[string]any{
					"do": "test",
				},
			},
		},
	}, map[string]AckMetadata{
		"1-0": {
			EventID: "1-0",
			Records: []AckRecord{
				{
					Group:     "workers",
					Consumer:  "worker-1",
					Timestamp: "2026-05-11T12:00:00Z",
				},
			},
		},
	}, []string{"workers", "billing"})

	wantHeaders := []string{"source", "status", "data"}
	if len(table.Headers) != len(wantHeaders) {
		t.Fatalf("headers = %+v, want %+v", table.Headers, wantHeaders)
	}
	for i := range wantHeaders {
		if table.Headers[i] != wantHeaders[i] {
			t.Fatalf("headers = %+v, want %+v", table.Headers, wantHeaders)
		}
	}

	if got := table.Rows[0].Values["source"]; got != "manual" {
		t.Fatalf("source cell = %q, want %q", got, "manual")
	}
	if got := table.Rows[0].Values["status"]; got != "created" {
		t.Fatalf("status cell = %q, want %q", got, "created")
	}
	if got := table.Rows[0].Values["data"]; got != "{\n  \"do\": \"test\"\n}" {
		t.Fatalf("data cell = %q, want pretty JSON object", got)
	}
	if got := table.Rows[0].AckLabel; got != "Acked 1/2" {
		t.Fatalf("ack label = %q, want Acked 1/2", got)
	}
	if got := table.Rows[0].AckState; got != "partial" {
		t.Fatalf("ack state = %q, want partial", got)
	}
	if got := table.Rows[0].AckTooltip; got != "Processed by\nworkers: worker-1\nAcked at 2026-05-11T12:00:00Z\n\nStill pending\nbilling" {
		t.Fatalf("ack tooltip = %q", got)
	}
}

func TestBuildAckMetadataParsesGroupedAckHash(t *testing.T) {
	acks := buildAckMetadata(map[string]string{
		"1-0:group-a": `{"consumer":"worker-1","timestamp":"2026-05-11T12:00:00Z"}`,
		"1-0:group-b": `{"consumer":"worker-2","timestamp":"2026-05-11T12:01:00Z"}`,
		"2-0:group-a": `{"group":"override","consumer":"worker-3","timestamp":"2026-05-11T12:02:00Z"}`,
	})

	if len(acks["1-0"].Records) != 2 {
		t.Fatalf("record count = %d, want 2", len(acks["1-0"].Records))
	}
	if got := acks["1-0"].Records[0].Group; got != "group-a" {
		t.Fatalf("group = %q, want group-a", got)
	}
	if got := acks["1-0"].Records[0].Consumer; got != "worker-1" {
		t.Fatalf("consumer = %q, want worker-1", got)
	}
	if got := acks["1-0"].Records[0].Timestamp; got != "2026-05-11T12:00:00Z" {
		t.Fatalf("timestamp = %q, want 2026-05-11T12:00:00Z", got)
	}
	if got := acks["2-0"].EventID; got != "2-0" {
		t.Fatalf("event id = %q, want 2-0", got)
	}
	if got := acks["2-0"].Records[0].Group; got != "override" {
		t.Fatalf("group = %q, want override", got)
	}
}

func TestBuildEventTableMarksCompleteWhenAllGroupsAcked(t *testing.T) {
	table := buildEventTable([]redis.XMessage{
		{ID: "1-0", Values: map[string]any{"source": "manual"}},
	}, map[string]AckMetadata{
		"1-0": {
			EventID: "1-0",
			Records: []AckRecord{
				{Group: "group-a", Consumer: "worker-1", Timestamp: "2026-05-11T12:00:00Z"},
				{Group: "group-b", Consumer: "worker-2", Timestamp: "2026-05-11T12:01:00Z"},
			},
		},
	}, []string{"group-a", "group-b"})

	if got := table.Rows[0].AckLabel; got != "Acked 2/2" {
		t.Fatalf("ack label = %q, want Acked 2/2", got)
	}
	if got := table.Rows[0].AckState; got != "complete" {
		t.Fatalf("ack state = %q, want complete", got)
	}
}
