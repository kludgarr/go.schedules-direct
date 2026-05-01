package schedulesdirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type lineupCapture struct {
	method     string
	path       string
	token      string
	verboseMap string
}

func lineupServer(t *testing.T, fixturePath string) (*httptest.Server, *lineupCapture) {
	t.Helper()
	cap := &lineupCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.token = r.Header.Get("token")
		cap.verboseMap = r.Header.Get("verboseMap")
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

func newAuthedClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	c, err := NewClient(testUserAgent,
		WithBaseURL(baseURL),
		WithToken("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestGetLineups(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/get_subscribed.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetLineups(context.Background())
	if err != nil {
		t.Fatalf("GetLineups: %v", err)
	}
	if cap.method != http.MethodGet || cap.path != "/lineups" {
		t.Errorf("request = %s %s", cap.method, cap.path)
	}
	if cap.token == "" {
		t.Error("token header missing")
	}
	if len(out.Lineups) != 4 {
		t.Fatalf("len(Lineups) = %d, want 4", len(out.Lineups))
	}
	if out.Lineups[0].Lineup != "USA-EXAMPLE-A" {
		t.Errorf("Lineups[0].Lineup = %q", out.Lineups[0].Lineup)
	}
	if out.Lineups[0].Transport != "IPTV" {
		t.Errorf("Transport = %q", out.Lineups[0].Transport)
	}
}

func TestGetLineupMapping(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/mapping.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetLineupMapping(context.Background(), "USA-OTA-90210", false)
	if err != nil {
		t.Fatalf("GetLineupMapping: %v", err)
	}
	if cap.path != "/lineups/USA-OTA-90210" {
		t.Errorf("path = %s", cap.path)
	}
	if cap.verboseMap != "" {
		t.Errorf("verboseMap header set when verbose=false: %q", cap.verboseMap)
	}
	if len(out.Map) != 3 {
		t.Errorf("len(Map) = %d, want 3", len(out.Map))
	}
	if len(out.Stations) != 3 {
		t.Errorf("len(Stations) = %d, want 3", len(out.Stations))
	}
	if out.Metadata.Lineup != "USA-OTA-90210" {
		t.Errorf("Metadata.Lineup = %q", out.Metadata.Lineup)
	}
	if out.Metadata.Transport != "Antenna" {
		t.Errorf("Metadata.Transport = %q", out.Metadata.Transport)
	}

	// Spot-check ATSC channel-map shape
	first := out.Map[0]
	if first.StationID != "10000001" || first.ATSCMajor != 2 || first.ATSCMinor != 1 {
		t.Errorf("first ChannelMap = %+v", first)
	}

	// Spot-check Station shape — multi-logo array, broadcaster, language array
	st := out.Stations[0]
	if st.StationID != "10000001" || st.Callsign != "EXMPDT" {
		t.Errorf("first Station = %+v", st)
	}
	if len(st.StationLogo) < 2 {
		t.Errorf("StationLogo entries = %d, want >=2", len(st.StationLogo))
	}
	// Logo should carry both md5 and hash (synthesis-gap field)
	if st.StationLogo[0].MD5 == "" || st.StationLogo[0].Hash == "" {
		t.Errorf("StationLogo missing md5/hash: %+v", st.StationLogo[0])
	}
	if st.Broadcaster == nil || st.Broadcaster.City != "Example City" {
		t.Errorf("Broadcaster = %+v", st.Broadcaster)
	}
}

func TestGetLineupMapping_VerboseSendsHeader(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/mapping.json")
	c := newAuthedClient(t, server.URL+"/")

	if _, err := c.GetLineupMapping(context.Background(), "USA-OTA-90210", true); err != nil {
		t.Fatalf("GetLineupMapping: %v", err)
	}
	if cap.verboseMap != "true" {
		t.Errorf("verboseMap header = %q, want true", cap.verboseMap)
	}
}

func TestAddLineup(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/add_response.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.AddLineup(context.Background(), "USA-OTA-90210")
	if err != nil {
		t.Fatalf("AddLineup: %v", err)
	}
	if cap.method != http.MethodPut {
		t.Errorf("method = %s, want PUT", cap.method)
	}
	if cap.path != "/lineups/USA-OTA-90210" {
		t.Errorf("path = %s", cap.path)
	}
	if out.Code != 0 {
		t.Errorf("Code = %d", out.Code)
	}
	if out.ChangesRemaining != 4 {
		t.Errorf("ChangesRemaining = %d, want 4", out.ChangesRemaining)
	}
	if !strings.Contains(out.Message, "Added") {
		t.Errorf("Message = %q", out.Message)
	}
}

func TestDeleteLineup(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/delete_response.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.DeleteLineup(context.Background(), "USA-OTA-90210")
	if err != nil {
		t.Fatalf("DeleteLineup: %v", err)
	}
	if cap.method != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", cap.method)
	}
	if out.ChangesRemaining != 5 {
		t.Errorf("ChangesRemaining = %d, want 5", out.ChangesRemaining)
	}
	if !strings.Contains(out.Message, "Deleted") {
		t.Errorf("Message = %q", out.Message)
	}
}

func TestPreviewLineup(t *testing.T) {
	server, cap := lineupServer(t, "testdata/lineup/preview.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.PreviewLineup(context.Background(), "USA-PEACOCK-X")
	if err != nil {
		t.Fatalf("PreviewLineup: %v", err)
	}
	if cap.path != "/lineups/preview/USA-PEACOCK-X" {
		t.Errorf("path = %s", cap.path)
	}
	if len(out) == 0 {
		t.Fatal("no preview channels")
	}
	if out[0].Channel == "" || out[0].Name == "" {
		t.Errorf("first preview entry incomplete: %+v", out[0])
	}
}
