package schedulesdirect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func availableServer(t *testing.T, fixturePath string) (*httptest.Server, *struct{ method, path string }) {
	t.Helper()
	cap := &struct{ method, path string }{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
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

func TestGetAvailableServices(t *testing.T) {
	server, cap := availableServer(t, "testdata/available/services.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetAvailableServices(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableServices: %v", err)
	}
	if cap.path != "/available" {
		t.Errorf("path = %s", cap.path)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	want := map[string]string{
		"COUNTRIES": "/20141201/available/countries",
		"LANGUAGES": "/20141201/available/languages",
		"DVB-T":     "/20141201/transmitters/{ISO 3166-1 alpha-3}",
	}
	for _, svc := range out {
		if w, ok := want[svc.Type]; !ok {
			t.Errorf("unexpected service Type %q", svc.Type)
		} else if svc.URI != w {
			t.Errorf("service %s URI = %q, want %q", svc.Type, svc.URI, w)
		}
	}
}

func TestGetAvailableCountries(t *testing.T) {
	server, cap := availableServer(t, "testdata/available/countries.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetAvailableCountries(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableCountries: %v", err)
	}
	if cap.path != "/available/countries" {
		t.Errorf("path = %s", cap.path)
	}
	if len(out) == 0 {
		t.Fatal("no regions returned")
	}
	carib, ok := out["Caribbean"]
	if !ok {
		t.Fatal(`expected "Caribbean" region`)
	}
	if len(carib) == 0 {
		t.Fatal("Caribbean has no countries")
	}
	// Spot-check one entry's shape.
	first := carib[0]
	if first.FullName == "" || first.ShortName == "" {
		t.Errorf("first Caribbean entry missing names: %+v", first)
	}
}

func TestGetAvailableLanguages(t *testing.T) {
	server, cap := availableServer(t, "testdata/available/languages.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetAvailableLanguages(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableLanguages: %v", err)
	}
	if cap.path != "/available/languages" {
		t.Errorf("path = %s", cap.path)
	}
	if got := out["en"]; got == "" {
		// "en" should be present in any sane languages map.
		t.Errorf("expected en mapping, languages map size = %d", len(out))
	}
}

func TestGetAvailableDVBS_NoCoverage(t *testing.T) {
	// Real wire for a non-UK account: empty array.
	server, cap := availableServer(t, "testdata/available/dvb-s.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetAvailableDVBS(context.Background())
	if err != nil {
		t.Fatalf("GetAvailableDVBS: %v", err)
	}
	if cap.path != "/available/dvb-s" {
		t.Errorf("path = %s", cap.path)
	}
	if len(out) != 0 {
		t.Errorf("len = %d, want 0 for non-UK account", len(out))
	}
}

func TestGetTransmitters_GBR(t *testing.T) {
	server, cap := availableServer(t, "testdata/transmitters/gbr.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetTransmitters(context.Background(), "GBR")
	if err != nil {
		t.Fatalf("GetTransmitters: %v", err)
	}
	if cap.path != "/transmitters/GBR" {
		t.Errorf("path = %s", cap.path)
	}
	if len(out) == 0 {
		t.Fatal("expected transmitters")
	}
	if got := out["Crystal Palace"]; got == "" {
		t.Errorf("expected Crystal Palace, got map = %v", out)
	}
}

func TestGetTransmitters_InvalidCountry(t *testing.T) {
	server, _ := availableServer(t, "testdata/transmitters/invalid_country.json")
	c := newTestClient(t, server.URL+"/")

	out, err := c.GetTransmitters(context.Background(), "USA")
	if out != nil {
		t.Fatalf("expected nil map on INVALID_PARAMETER:COUNTRY, got %v", out)
	}
	var sdErr *Error
	if !errors.As(err, &sdErr) {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if sdErr.Code != CodeInvalidParameterCountry {
		t.Errorf("Code = %d, want %d", sdErr.Code, CodeInvalidParameterCountry)
	}
}
