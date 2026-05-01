package schedulesdirect

import (
	"errors"
	"net/http"
	"testing"
)

func TestNewClient_RequiresUserAgent(t *testing.T) {
	if _, err := NewClient(""); !errors.Is(err, ErrMissingUserAgent) {
		t.Fatalf("expected ErrMissingUserAgent, got %v", err)
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c, err := NewClient("test/1.0")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.UserAgent != "test/1.0" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "test/1.0")
	}
	if c.BaseURL != BaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, BaseURL)
	}
	if c.HTTPClient != http.DefaultClient {
		t.Errorf("HTTPClient = %v, want http.DefaultClient", c.HTTPClient)
	}
	if c.Token != "" {
		t.Errorf("Token = %q, want empty", c.Token)
	}
}

func TestNewClient_Options(t *testing.T) {
	hc := &http.Client{}
	c, err := NewClient(
		"test/1.0",
		WithHTTPClient(hc),
		WithBaseURL("https://example.test/v1/"),
		WithToken("abc123"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.HTTPClient != hc {
		t.Error("WithHTTPClient did not take effect")
	}
	if c.BaseURL != "https://example.test/v1/" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.Token != "abc123" {
		t.Errorf("Token = %q", c.Token)
	}
}
