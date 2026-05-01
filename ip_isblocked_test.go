package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func ipBlockServer(t *testing.T, fixture string) (*httptest.Server, *struct{ method, path, userAgent string }) {
	t.Helper()
	cap := &struct{ method, path, userAgent string }{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.userAgent = r.Header.Get("User-Agent")
		data, err := os.ReadFile("testdata/ip_isblocked/" + fixture)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixture, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, cap
}

func TestGetIPBlockStatus_NotBlocked(t *testing.T) {
	server, cap := ipBlockServer(t, "not_blocked.json")
	c := newTestClient(t, server.URL+"/")

	s, err := c.GetIPBlockStatus(context.Background())
	if err != nil {
		t.Fatalf("GetIPBlockStatus: %v", err)
	}
	if cap.method != http.MethodGet {
		t.Errorf("method = %s", cap.method)
	}
	if cap.path != "/ip_isblocked" {
		t.Errorf("path = %s", cap.path)
	}
	if s.Code != 0 {
		t.Errorf("Code = %d", s.Code)
	}
	if s.BlockedOnLoadBalancer != false {
		t.Errorf("BlockedOnLoadBalancer = %v, want false", s.BlockedOnLoadBalancer)
	}
}

func TestGetIPBlockStatus_Blocked(t *testing.T) {
	server, _ := ipBlockServer(t, "blocked.json")
	c := newTestClient(t, server.URL+"/")

	s, err := c.GetIPBlockStatus(context.Background())
	if err != nil {
		t.Fatalf("GetIPBlockStatus: %v", err)
	}
	if s.BlockedOnLoadBalancer != true {
		t.Errorf("BlockedOnLoadBalancer = %v, want true", s.BlockedOnLoadBalancer)
	}
}
