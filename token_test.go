package schedulesdirect

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	// sha1("password") = 5baa61e4c9b93f3f0682250b6cf8331b7ee68fd8
	got := HashPassword("password")
	want := "5baa61e4c9b93f3f0682250b6cf8331b7ee68fd8"
	if got != want {
		t.Errorf("HashPassword(\"password\") = %q, want %q", got, want)
	}
	if len(got) != 40 {
		t.Errorf("HashPassword length = %d, want 40", len(got))
	}
}

// tokenServer wires an httptest.Server to a fixture file under testdata/token/.
// It also captures the request for assertions.
type tokenCapture struct {
	method      string
	path        string
	userAgent   string
	contentType string
	body        []byte
}

func tokenServer(t *testing.T, fixture string) (*httptest.Server, *tokenCapture) {
	t.Helper()
	cap := &tokenCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.userAgent = r.Header.Get("User-Agent")
		cap.contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		cap.body = body

		data, err := os.ReadFile("testdata/token/" + fixture)
		if err != nil {
			t.Fatalf("read fixture %q: %v", fixture, err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	return server, cap
}

// testUserAgent borrows the example-client UA shape (real product, real-format wire)
// rather than synthesizing a fake "test/1.0" — keeps test request shape close
// to actual SD consumer traffic.
const testUserAgent = "example-client/1.0"

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(testUserAgent, WithBaseURL(baseURL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetToken_Success(t *testing.T) {
	server, cap := tokenServer(t, "success.json")
	c := newTestClient(t, server.URL+"/")

	tok, err := c.GetToken(context.Background(), Account{
		Username: "user@example.com",
		Password: HashPassword("password"),
	})
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}

	// Request shape
	if cap.method != http.MethodPost {
		t.Errorf("method = %s, want POST", cap.method)
	}
	if cap.path != "/token" {
		t.Errorf("path = %s, want /token", cap.path)
	}
	if cap.userAgent != testUserAgent {
		t.Errorf("User-Agent = %q, want %q", cap.userAgent, testUserAgent)
	}
	if !strings.HasPrefix(cap.contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", cap.contentType)
	}
	var sentAccount Account
	if err := json.Unmarshal(cap.body, &sentAccount); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if sentAccount.Username != "user@example.com" {
		t.Errorf("sent username = %q", sentAccount.Username)
	}
	if sentAccount.NewToken != false {
		t.Errorf("default NewToken should be false (omitted)")
	}

	// Response shape — values mirror the real wire sample at
	// epg-sd-eval/sample_data/token.json. Note that real /token success wire
	// does NOT carry a "response" field; only "message" is populated.
	if tok.Code != 0 {
		t.Errorf("Code = %d, want 0", tok.Code)
	}
	if tok.Message != "OK" {
		t.Errorf("Message = %q, want OK", tok.Message)
	}
	if tok.Token != "0123456789abcdef0123456789abcdef" {
		t.Errorf("Token = %q", tok.Token)
	}
	if len(tok.Token) != 32 {
		t.Errorf("Token length = %d, want 32", len(tok.Token))
	}
	if tok.TokenExpires != 4070995200 {
		t.Errorf("TokenExpires = %d", tok.TokenExpires)
	}
	if tok.ServerTime != 4070908800 {
		t.Errorf("ServerTime = %d", tok.ServerTime)
	}
}

func TestGetToken_NewTokenFlag(t *testing.T) {
	server, cap := tokenServer(t, "success.json")
	c := newTestClient(t, server.URL+"/")

	_, err := c.GetToken(context.Background(), Account{
		Username: "user@example.com",
		Password: HashPassword("password"),
		NewToken: true,
	})
	if err != nil {
		t.Fatalf("GetToken: %v", err)
	}
	var sent Account
	if err := json.Unmarshal(cap.body, &sent); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if sent.NewToken != true {
		t.Errorf("NewToken = %v in request body, want true", sent.NewToken)
	}
}

func TestGetToken_ServiceOffline(t *testing.T) {
	server, _ := tokenServer(t, "service_offline.json")
	c := newTestClient(t, server.URL+"/")

	tok, err := c.GetToken(context.Background(), Account{
		Username: "user@example.com",
		Password: HashPassword("password"),
	})
	if tok != nil {
		t.Fatalf("expected nil Token on SERVICE_OFFLINE, got %+v", tok)
	}

	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeServiceOffline {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeServiceOffline)
	}
	if sdErr.Response != "SERVICE_OFFLINE" {
		t.Errorf("Response = %q", sdErr.Response)
	}
	if sdErr.Token != "CAFEDEADBEEFCAFEDEADBEEFCAFEDEADBEEFCAFE" {
		t.Errorf("placeholder token = %q", sdErr.Token)
	}
	if sdErr.TokenExpires != 0 {
		t.Errorf("TokenExpires = %d, want 0 on SERVICE_OFFLINE", sdErr.TokenExpires)
	}
}

func TestGetToken_InvalidUser(t *testing.T) {
	server, _ := tokenServer(t, "invalid_user.json")
	c := newTestClient(t, server.URL+"/")

	tok, err := c.GetToken(context.Background(), Account{
		Username: "user@example.com",
		Password: HashPassword("wrong"),
	})
	if tok != nil {
		t.Fatalf("expected nil Token on INVALID_USER, got %+v", tok)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeInvalidUser {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeInvalidUser)
	}
	if sdErr.Response != "INVALID_USER" {
		t.Errorf("Response = %q", sdErr.Response)
	}
}

func TestError_ErrorString(t *testing.T) {
	e := &Error{
		BaseResponse: BaseResponse{
			Code:     CodeServiceOffline,
			Response: "SERVICE_OFFLINE",
			Message:  "Server offline for maintenance.",
		},
	}
	got := e.Error()
	if !strings.Contains(got, "3000") || !strings.Contains(got, "SERVICE_OFFLINE") {
		t.Errorf("Error() = %q, missing code or response", got)
	}
}
