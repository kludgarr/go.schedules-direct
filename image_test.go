package schedulesdirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetImageURL_303Redirect(t *testing.T) {
	const expectedLocation = "https://schedulesdirect-api20141201a.s3.amazonaws.com/a/abc.jpg?X-Amz-Signature=xxxx"
	var capturedPath string
	var capturedToken string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedToken = r.Header.Get("token")
		w.Header().Set("Location", expectedLocation)
		w.WriteHeader(http.StatusSeeOther)
	}))
	t.Cleanup(server.Close)

	c := newAuthedClient(t, server.URL+"/")
	got, err := c.GetImageURL(context.Background(), "abc.jpg")
	if err != nil {
		t.Fatalf("GetImageURL: %v", err)
	}
	if got != expectedLocation {
		t.Errorf("URL = %q, want %q", got, expectedLocation)
	}
	if capturedPath != "/image/abc.jpg" {
		t.Errorf("path = %s", capturedPath)
	}
	if capturedToken == "" {
		t.Error("token header missing")
	}
}

func TestGetImageURL_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"response":"IMAGE_NOT_FOUND","code":5000,"message":"Could not find requested image."}`))
	}))
	t.Cleanup(server.Close)

	c := newAuthedClient(t, server.URL+"/")
	url, err := c.GetImageURL(context.Background(), "missing.jpg")
	if url != "" {
		t.Fatalf("expected empty URL on IMAGE_NOT_FOUND, got %q", url)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeImageNotFound {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeImageNotFound)
	}
}

func TestGetImageURL_QuotaExceeded(t *testing.T) {
	// 5002 MAX_IMAGE_DOWNLOADS — consumer must stop on this.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"response":"MAX_IMAGE_DOWNLOADS","code":5002,"message":"Maximum image downloads reached."}`))
	}))
	t.Cleanup(server.Close)

	c := newAuthedClient(t, server.URL+"/")
	_, err := c.GetImageURL(context.Background(), "any.jpg")
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeMaxImageDownloads {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeMaxImageDownloads)
	}
}
