package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func celebrityServer(t *testing.T, fixturePath string) *httptest.Server {
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

func TestGetCelebrity(t *testing.T) {
	server := celebrityServer(t, "testdata/celebrity/celebrity_39294.json")
	c := newAuthedClient(t, server.URL+"/")

	celeb, err := c.GetCelebrity(context.Background(), "39294")
	if err != nil {
		t.Fatalf("GetCelebrity: %v", err)
	}
	if celeb.PersonID != "39294" {
		t.Errorf("PersonID = %q", celeb.PersonID)
	}
	if celeb.Type != "Person" {
		t.Errorf("Type = %q", celeb.Type)
	}
	if len(celeb.Names) == 0 || !celeb.Names[0].IsPrimary {
		t.Errorf("Names = %+v", celeb.Names)
	}
	if len(celeb.Mediography) == 0 {
		t.Errorf("Mediography empty")
	}
}

func TestGetCelebrityImages(t *testing.T) {
	server := celebrityServer(t, "testdata/celebrity/images_39294.json")
	c := newAuthedClient(t, server.URL+"/")

	imgs, err := c.GetCelebrityImages(context.Background(), "39294")
	if err != nil {
		t.Fatalf("GetCelebrityImages: %v", err)
	}
	if len(imgs) == 0 {
		t.Fatal("no images")
	}
	if imgs[0].URI == "" {
		t.Errorf("first image URI empty: %+v", imgs[0])
	}
}
