package schedulesdirect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type programsCapture struct {
	method string
	path   string
	token  string
	body   []byte
}

func programsServer(t *testing.T, fixturePath string) (*httptest.Server, *programsCapture) {
	t.Helper()
	cap := &programsCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.token = r.Header.Get("token")
		body, _ := io.ReadAll(r.Body)
		cap.body = body
		data, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixturePath, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, cap
}

func TestGetPrograms(t *testing.T) {
	server, cap := programsServer(t, "testdata/programs/batch.json")
	c := newAuthedClient(t, server.URL+"/")

	ids := []string{"EP012801050074", "MV008961210000"}
	out, err := c.GetPrograms(context.Background(), ids)
	if err != nil {
		t.Fatalf("GetPrograms: %v", err)
	}

	// Request shape
	if cap.method != http.MethodPost || cap.path != "/programs" {
		t.Errorf("request = %s %s", cap.method, cap.path)
	}
	var sentIDs []string
	if err := json.Unmarshal(cap.body, &sentIDs); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}
	if len(sentIDs) != 2 || sentIDs[0] != "EP012801050074" {
		t.Errorf("sent IDs = %v", sentIDs)
	}

	// Response shape — fixture has 500 entries
	if len(out) != 500 {
		t.Errorf("len = %d, want 500", len(out))
	}

	// Spot-check the first entry — should be a fully populated success
	first := out[0]
	if first.ProgramID == "" {
		t.Errorf("ProgramID empty: %+v", first)
	}
	if first.Code != 0 {
		t.Errorf("first entry has Code = %d (expected success)", first.Code)
	}
	if len(first.Titles) == 0 {
		t.Errorf("Titles empty for %s", first.ProgramID)
	}
	if first.Titles[0].Title120 == "" {
		t.Errorf("Title120 empty: %+v", first.Titles[0])
	}

	// Find an entry with cast and verify shape
	var withCast *Program
	for i := range out {
		if len(out[i].Cast) > 0 {
			withCast = &out[i]
			break
		}
	}
	if withCast == nil {
		t.Fatal("no entry has cast (unexpected for sample)")
	}
	c0 := withCast.Cast[0]
	if c0.Name == "" || c0.Role == "" || c0.BillingOrder == "" {
		t.Errorf("first cast member missing required fields: %+v", c0)
	}

	// Find an entry with metadata and verify Gracenote/TVmaze shape
	var withMetadata *Program
	for i := range out {
		if len(out[i].Metadata) > 0 {
			withMetadata = &out[i]
			break
		}
	}
	if withMetadata == nil {
		t.Fatal("no entry has metadata")
	}
	hasGracenote, hasTVmaze := false, false
	for _, m := range withMetadata.Metadata {
		if m.Gracenote != nil {
			hasGracenote = true
		}
		if m.TVmaze != nil {
			hasTVmaze = true
		}
	}
	if !hasGracenote && !hasTVmaze {
		t.Errorf("metadata entries had neither Gracenote nor TVmaze populated: %+v", withMetadata.Metadata)
	}

	// Verify Description1000 parses
	var withDesc *Program
	for i := range out {
		if len(out[i].Descriptions.Description1000) > 0 {
			withDesc = &out[i]
			break
		}
	}
	if withDesc == nil {
		t.Skip("no Description1000 in sample")
	}
	d := withDesc.Descriptions.Description1000[0]
	if d.Description == "" || d.DescriptionLanguage == "" {
		t.Errorf("DescriptionEntry missing fields: %+v", d)
	}
}
