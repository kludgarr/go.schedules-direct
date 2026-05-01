package schedulesdirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type statusCapture struct {
	method    string
	path      string
	userAgent string
	token     string
}

func statusServer(t *testing.T, fixture string) (*httptest.Server, *statusCapture) {
	t.Helper()
	cap := &statusCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.userAgent = r.Header.Get("User-Agent")
		cap.token = r.Header.Get("token")

		data, err := os.ReadFile("testdata/status/" + fixture)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixture, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, cap
}

func TestGetStatus_Success(t *testing.T) {
	server, cap := statusServer(t, "success.json")
	c, err := NewClient(testUserAgent,
		WithBaseURL(server.URL+"/"),
		WithToken("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	s, err := c.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	// Request shape
	if cap.method != http.MethodGet {
		t.Errorf("method = %s", cap.method)
	}
	if cap.path != "/status" {
		t.Errorf("path = %s", cap.path)
	}
	if cap.userAgent != testUserAgent {
		t.Errorf("UA = %q", cap.userAgent)
	}
	if cap.token != "0123456789abcdef0123456789abcdef" {
		t.Errorf("token header = %q", cap.token)
	}

	// Response shape
	if s.Account.MaxLineups != 4 {
		t.Errorf("MaxLineups = %d", s.Account.MaxLineups)
	}
	if s.Account.AccountExpiration != 4070908800 {
		t.Errorf("AccountExpiration = %d", s.Account.AccountExpiration)
	}
	if len(s.Lineups) != 4 {
		t.Fatalf("len(Lineups) = %d, want 4", len(s.Lineups))
	}
	if s.Lineups[0].Lineup != "USA-EXAMPLE-A" {
		t.Errorf("Lineups[0].Lineup = %q", s.Lineups[0].Lineup)
	}
	if s.Lineups[0].URI != "/20141201/lineups/USA-EXAMPLE-A" {
		t.Errorf("Lineups[0].URI = %q", s.Lineups[0].URI)
	}
	if len(s.SystemStatus) != 1 {
		t.Fatalf("len(SystemStatus) = %d, want 1", len(s.SystemStatus))
	}
	if s.SystemStatus[0].Status != "Online" {
		t.Errorf("SystemStatus[0].Status = %q", s.SystemStatus[0].Status)
	}
	if s.TokenExpires != 4070995200 {
		t.Errorf("TokenExpires = %d", s.TokenExpires)
	}
	if s.ServerTime != 4070908800 {
		t.Errorf("ServerTime = %d", s.ServerTime)
	}
}

func TestGetStatus_NoTokenSendsNoHeader(t *testing.T) {
	server, cap := statusServer(t, "token_missing.json")
	c, err := NewClient(testUserAgent, WithBaseURL(server.URL+"/"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	s, err := c.GetStatus(context.Background())
	if s != nil {
		t.Fatalf("expected nil Status on TOKEN_MISSING, got %+v", s)
	}
	if cap.token != "" {
		t.Errorf("token header should be empty when c.Token is empty, got %q", cap.token)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeTokenMissing {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeTokenMissing)
	}
}

func TestGetStatus_TokenExpired(t *testing.T) {
	server, _ := statusServer(t, "token_expired.json")
	c, err := NewClient(testUserAgent,
		WithBaseURL(server.URL+"/"),
		WithToken("expired-token"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	s, err := c.GetStatus(context.Background())
	if s != nil {
		t.Fatalf("expected nil Status on TOKEN_EXPIRED, got %+v", s)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeTokenExpired {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeTokenExpired)
	}
}
