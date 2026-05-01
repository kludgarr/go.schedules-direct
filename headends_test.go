package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type headendsCapture struct {
	method string
	path   string
	query  string
	token  string
}

func headendsServer(t *testing.T, fixturePath string) (*httptest.Server, *headendsCapture) {
	t.Helper()
	cap := &headendsCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.query = r.URL.RawQuery
		cap.token = r.Header.Get("token")
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

func TestGetHeadends(t *testing.T) {
	server, cap := headendsServer(t, "testdata/headends/usa.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetHeadends(context.Background(), "USA", "90210")
	if err != nil {
		t.Fatalf("GetHeadends: %v", err)
	}
	if cap.method != http.MethodGet {
		t.Errorf("method = %s", cap.method)
	}
	if cap.path != "/headends" {
		t.Errorf("path = %s", cap.path)
	}
	if cap.query != "country=USA&postalcode=90210" {
		t.Errorf("query = %s", cap.query)
	}
	if cap.token == "" {
		t.Error("token header missing")
	}
	if len(out) == 0 {
		t.Fatal("no headends returned")
	}
	if out[0].Headend == "" {
		t.Errorf("first Headend = %+v (empty Headend field)", out[0])
	}
	if len(out[0].Lineups) == 0 {
		t.Errorf("first Headend has no lineups: %+v", out[0])
	}
}
