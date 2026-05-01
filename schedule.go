package schedulesdirect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ScheduleRequest is one entry in the request body of POST /schedules and
// POST /schedules/md5.
//
// StationID identifies the station; Date selects specific dates (YYYY-MM-DD).
// Omit Date to get all available dates SD has for the station.
type ScheduleRequest struct {
	StationID string   `json:"stationID"`
	Date      []string `json:"date,omitempty"`
}

// ScheduleMd5Entry is one entry in the GET /schedules/md5 response —
// keyed by stationID then by date.
//
// Hash is the 32-character standard MD5 hex; MD5 is the 22-character base64
// legacy form. SD has signaled hash will become mandatory and md5 will be
// removed in a future API version (rkulagow forum post 2023-09-11). Consumers
// should prefer Hash for change detection.
type ScheduleMd5Entry struct {
	Code         int    `json:"code,omitempty"`
	Message      string `json:"message,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Hash         string `json:"hash,omitempty"`
	MD5          string `json:"md5,omitempty"`
}

// StationSchedule is one entry in the POST /schedules response array.
//
// Two shapes share this struct:
//   - Success: Programs is populated, Metadata.MD5/Modified set, Code is 0.
//   - Per-station failure: Code != 0, Response carries the SD code label, and
//     auxiliary fields (RetryTime, MinDate/MaxDate/RequestedDate) are populated
//     per the failure type (SCHEDULE_QUEUED 7100, SCHEDULE_RANGE_EXCEEDED 7020,
//     STATIONID_DELETED 2201, etc.). Programs may be empty in this case.
//
// Consumers iterate the response array and check Code per entry; per-station
// failures do NOT cause the top-level call to return an *Error. Top-level
// envelope errors (token missing, invalid request body) still come back via
// the usual error path.
type StationSchedule struct {
	StationID string             `json:"stationID"`
	Programs  []ScheduledProgram `json:"programs,omitempty"`
	Metadata  ScheduleMetadata   `json:"metadata,omitempty"`

	// Per-station failure fields. Populated when Code != 0.
	Code          int    `json:"code,omitempty"`
	Response      string `json:"response,omitempty"`
	ServerID      string `json:"serverID,omitempty"`
	Message       string `json:"message,omitempty"`
	RetryTime     string `json:"retryTime,omitempty"`
	MinDate       string `json:"minDate,omitempty"`
	MaxDate       string `json:"maxDate,omitempty"`
	RequestedDate string `json:"requestedDate,omitempty"`
}

// ScheduledProgram is one program-slot entry within StationSchedule.Programs.
//
// videoProperties wire values exceed the OpenAPI spec's documented enum (e.g.
// "HD 1080i" is observed in wire but not in the spec enum [3d, enhanced, hdtv,
// hdr, letterbox, sdtv, uhdtv]); the library models as []string with no
// enforcement (synthesis gap; wire is canonical).
type ScheduledProgram struct {
	ProgramID          string           `json:"programID"`
	AirDateTime        string           `json:"airDateTime"`
	Duration           int              `json:"duration"`
	MD5                string           `json:"md5,omitempty"`
	Hash               string           `json:"hash,omitempty"`
	New                bool             `json:"new,omitempty"`
	LiveTapeDelay      string           `json:"liveTapeDelay,omitempty"`
	IsPremiereOrFinale string           `json:"isPremiereOrFinale,omitempty"`
	AudioProperties    []string         `json:"audioProperties,omitempty"`
	VideoProperties    []string         `json:"videoProperties,omitempty"`
	Ratings            []ScheduleRating `json:"ratings,omitempty"`
	Multipart          *Multipart       `json:"multipart,omitempty"`
	CableInTheClassroom bool            `json:"cableInTheClassroom,omitempty"`
	Catchup            bool             `json:"catchup,omitempty"`
	Continued          bool             `json:"continued,omitempty"`
	Educational        bool             `json:"educational,omitempty"`
	JoinedInProgress   bool             `json:"joinedInProgress,omitempty"`
	LeftInProgress     bool             `json:"leftInProgress,omitempty"`
	Premiere           bool             `json:"premiere,omitempty"`
	ProgramBreak       bool             `json:"programBreak,omitempty"`
	Repeat             bool             `json:"repeat,omitempty"`
	Signed             bool             `json:"signed,omitempty"`
	SubjectToBlackout  bool             `json:"subjectToBlackout,omitempty"`
	TimeApproximate    bool             `json:"timeApproximate,omitempty"`
	Free               bool             `json:"free,omitempty"`
	SAPLanguage        string           `json:"SAPLanguage,omitempty"`
}

// ScheduleMetadata is the per-station metadata block in StationSchedule.
type ScheduleMetadata struct {
	Modified      string `json:"modified,omitempty"`
	ModifiedEpoch int64  `json:"modifiedEpoch,omitempty"`
	MD5           string `json:"md5,omitempty"`
	Hash          string `json:"hash,omitempty"`
	StartDate     string `json:"startDate,omitempty"`
	Code          int    `json:"code,omitempty"`
	IsDeleted     bool   `json:"isDeleted,omitempty"`
}

// ScheduleRating is one rating entry on a ScheduledProgram.
//
// SubRating extends Body/Code with a content-warning sub-axis (e.g.
// "Language|Dialog"); pipe-separated when multi-valued.
type ScheduleRating struct {
	Body      string `json:"body,omitempty"`
	Code      string `json:"code,omitempty"`
	SubRating string `json:"subRating,omitempty"`
}

// Multipart describes a multi-part program slot.
type Multipart struct {
	PartNumber int `json:"partNumber,omitempty"`
	TotalParts int `json:"totalParts,omitempty"`
}

// GetSchedules calls POST /schedules with the requested stations and dates,
// returning per-station schedule entries.
//
// Each StationSchedule may carry per-station failure (Code != 0); consumers
// iterate and dispatch on Code. Top-level envelope errors (e.g. token issues)
// surface as the returned error.
//
// Max 5000 ScheduleRequest entries per call (SD-imposed). Token-required.
func (c *Client) GetSchedules(ctx context.Context, requests []ScheduleRequest) ([]StationSchedule, error) {
	body, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal requests: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"schedules", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out []StationSchedule
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetSchedulesMd5 calls POST /schedules/md5 with the requested stations and
// dates, returning a 2-level map keyed by stationID then by date.
//
// Max 5000 ScheduleRequest entries per call (SD-imposed). Token-required.
func (c *Client) GetSchedulesMd5(ctx context.Context, requests []ScheduleRequest) (map[string]map[string]ScheduleMd5Entry, error) {
	body, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal requests: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"schedules/md5", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out map[string]map[string]ScheduleMd5Entry
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
