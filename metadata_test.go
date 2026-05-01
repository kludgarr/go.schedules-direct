package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func metadataServer(t *testing.T, fixturePath string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(fixturePath)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixturePath, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestGetProgramImages(t *testing.T) {
	server := metadataServer(t, "testdata/metadata/programs_single.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetProgramImages(context.Background(), "EP017598170269")
	if err != nil {
		t.Fatalf("GetProgramImages: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("no images")
	}
	first := out[0]
	if first.URI == "" {
		t.Errorf("first image URI empty: %+v", first)
	}
	if first.Category == "" {
		t.Errorf("first image Category empty: %+v", first)
	}
	if first.Width.String() == "" || first.Height.String() == "" {
		t.Errorf("first image Width/Height: %v / %v", first.Width, first.Height)
	}
}

func TestGetProgramImagesBatch(t *testing.T) {
	server := metadataServer(t, "testdata/metadata/programs_batch.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetProgramImagesBatch(context.Background(), []string{"EP017598170269"})
	if err != nil {
		t.Fatalf("GetProgramImagesBatch: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("empty batch response")
	}
	if out[0].ProgramID == "" {
		t.Errorf("first entry programID empty: %+v", out[0])
	}
	if len(out[0].Data) == 0 {
		t.Errorf("first entry data empty: %+v", out[0])
	}
}

func TestGetProgramDescriptions(t *testing.T) {
	server := metadataServer(t, "testdata/metadata/descriptions.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetProgramDescriptions(context.Background(), []string{"EP017598170269"})
	if err != nil {
		t.Fatalf("GetProgramDescriptions: %v", err)
	}
	desc, ok := out["SH017598170000"]
	if !ok {
		t.Fatal(`expected SH017598170000 key`)
	}
	if desc.Description1000 == "" {
		t.Errorf("Description1000 empty: %+v", desc)
	}
}
