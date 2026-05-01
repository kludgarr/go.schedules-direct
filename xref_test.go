package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestGetLanguageCrossReferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("testdata/xref/batch.json")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)

	c := newAuthedClient(t, server.URL+"/")
	out, err := c.GetLanguageCrossReferences(context.Background(), []string{"EP017598170269"})
	if err != nil {
		t.Fatalf("GetLanguageCrossReferences: %v", err)
	}
	xrefs, ok := out["EP017598170269"]
	if !ok {
		t.Fatal(`expected EP017598170269 key`)
	}
	if len(xrefs) == 0 {
		t.Fatal("empty xrefs")
	}
	if xrefs[0].DescriptionLanguage == "" {
		t.Errorf("DescriptionLanguage empty: %+v", xrefs[0])
	}
	if xrefs[0].Hash == "" || xrefs[0].MD5 == "" {
		t.Errorf("Hash/MD5 missing: %+v", xrefs[0])
	}
}
