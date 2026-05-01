package schedulesdirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// versionServer wires an httptest.Server to a fixture under testdata/version/
// and captures the request for assertions.
func versionServer(t *testing.T, fixture string) (*httptest.Server, *struct{ method, path, userAgent string }) {
	t.Helper()
	cap := &struct{ method, path, userAgent string }{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.userAgent = r.Header.Get("User-Agent")
		data, err := os.ReadFile("testdata/version/" + fixture)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixture, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, cap
}

func TestGetVersion_Success(t *testing.T) {
	server, cap := versionServer(t, "success.json")
	c := newTestClient(t, server.URL+"/")

	v, err := c.GetVersion(context.Background(), "example-client")
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if cap.method != http.MethodGet {
		t.Errorf("method = %s, want GET", cap.method)
	}
	if cap.path != "/version/example-client" {
		t.Errorf("path = %s, want /version/example-client", cap.path)
	}
	if cap.userAgent != testUserAgent {
		t.Errorf("UA = %q, want %q", cap.userAgent, testUserAgent)
	}
	if v.Code != 0 {
		t.Errorf("Code = %d, want 0", v.Code)
	}
	if v.Response != "OK" {
		t.Errorf("Response = %q, want OK", v.Response)
	}
	if v.Client != "example-client" {
		t.Errorf("Client = %q", v.Client)
	}
	if v.Version != "1.0" {
		t.Errorf("Version = %q", v.Version)
	}
}

func TestGetVersion_UnknownClient(t *testing.T) {
	server, _ := versionServer(t, "unknown_client.json")
	c := newTestClient(t, server.URL+"/")

	v, err := c.GetVersion(context.Background(), "no-such-client")
	if v != nil {
		t.Fatalf("expected nil Version on UNKNOWN_CLIENT, got %+v", v)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeUnknownClient {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeUnknownClient)
	}
}

func TestGetVersion_PathEscape(t *testing.T) {
	// Verify that names with chars needing URL escape round-trip correctly:
	// PathEscape encodes them, the server decodes them, the handler sees the
	// original semantic value.
	server, cap := versionServer(t, "success.json")
	c := newTestClient(t, server.URL+"/")

	if _, err := c.GetVersion(context.Background(), "client with spaces"); err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if cap.path != "/version/client with spaces" {
		t.Errorf("decoded path = %q, want %q", cap.path, "/version/client with spaces")
	}
}
