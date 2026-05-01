package schedulesdirect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type scheduleCapture struct {
	method      string
	path        string
	contentType string
	token       string
	body        []byte
}

func scheduleServer(t *testing.T, fixturePath string) (*httptest.Server, *scheduleCapture) {
	t.Helper()
	cap := &scheduleCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.contentType = r.Header.Get("Content-Type")
		cap.token = r.Header.Get("token")
		body, _ := io.ReadAll(r.Body)
		cap.body = body
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

func TestGetSchedules(t *testing.T) {
	server, cap := scheduleServer(t, "testdata/schedule/schedules.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetSchedules(context.Background(), []ScheduleRequest{
		{StationID: "10000001"},
	})
	if err != nil {
		t.Fatalf("GetSchedules: %v", err)
	}
	if cap.method != http.MethodPost || cap.path != "/schedules" {
		t.Errorf("request = %s %s", cap.method, cap.path)
	}
	if len(out) == 0 {
		t.Fatal("empty response")
	}

	first := out[0]
	if first.StationID == "" {
		t.Errorf("StationID empty: %+v", first)
	}
	if len(first.Programs) == 0 {
		t.Errorf("Programs empty for first station: %+v", first)
	}
	if first.Code != 0 {
		t.Errorf("first station Code = %d (treating as success)", first.Code)
	}
	if first.Metadata.MD5 == "" {
		t.Errorf("Metadata.MD5 empty: %+v", first.Metadata)
	}

	prog := first.Programs[0]
	if prog.ProgramID == "" {
		t.Errorf("ProgramID empty: %+v", prog)
	}
	if prog.AirDateTime == "" {
		t.Errorf("AirDateTime empty: %+v", prog)
	}
	if prog.Duration <= 0 {
		t.Errorf("Duration = %d", prog.Duration)
	}
	if prog.Hash == "" || prog.MD5 == "" {
		t.Errorf("Hash/MD5 missing: %+v", prog)
	}
}

func TestGetSchedulesMd5(t *testing.T) {
	server, cap := scheduleServer(t, "testdata/schedule/md5.json")
	c := newAuthedClient(t, server.URL+"/")

	out, err := c.GetSchedulesMd5(context.Background(), []ScheduleRequest{
		{StationID: "10000001", Date: []string{"2026-04-21", "2026-04-22"}},
		{StationID: "10000002"},
	})
	if err != nil {
		t.Fatalf("GetSchedulesMd5: %v", err)
	}

	// Request shape
	if cap.method != http.MethodPost {
		t.Errorf("method = %s", cap.method)
	}
	if cap.path != "/schedules/md5" {
		t.Errorf("path = %s", cap.path)
	}
	if cap.contentType != "application/json" {
		t.Errorf("Content-Type = %q", cap.contentType)
	}
	if cap.token == "" {
		t.Error("token header missing")
	}
	var sentReq []ScheduleRequest
	if err := json.Unmarshal(cap.body, &sentReq); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if len(sentReq) != 2 {
		t.Errorf("len(request) = %d, want 2", len(sentReq))
	}
	if sentReq[0].StationID != "10000001" || len(sentReq[0].Date) != 2 {
		t.Errorf("request[0] = %+v", sentReq[0])
	}
	if sentReq[1].StationID != "10000002" || len(sentReq[1].Date) != 0 {
		t.Errorf("request[1] = %+v (Date should be omitted)", sentReq[1])
	}

	// Response shape
	if len(out) == 0 {
		t.Fatal("empty response")
	}
	st, ok := out["10000001"]
	if !ok {
		t.Fatal(`stationID "10000001" missing from response`)
	}
	day, ok := st["2026-04-21"]
	if !ok {
		t.Fatal(`date "2026-04-21" missing for stationID "10000001"`)
	}
	if day.Code != 0 {
		t.Errorf("Code = %d, want 0", day.Code)
	}
	if day.Hash == "" {
		t.Errorf("Hash empty: %+v", day)
	}
	if len(day.Hash) != 32 {
		t.Errorf("Hash length = %d, want 32 (canonical hex)", len(day.Hash))
	}
	if day.MD5 == "" {
		t.Errorf("MD5 (legacy) empty: %+v", day)
	}
}
