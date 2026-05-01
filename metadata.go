package schedulesdirect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ProgramImage is one image-metadata entry returned from /metadata/programs/
// and /metadata/programs/{programId}.
//
// URIs in image responses are ephemeral — do not store them permanently. If
// URI is a fully-qualified S3 URL (contains "https://"), fetch it directly
// without GET /image/{uri}; otherwise resolve via GetImageURL.
//
// Width/Height are json.Number to accept both integer and stringified-integer
// forms SD's wire returns (legacy responses). Consumers call .Int64() or
// parse to int as needed.
type ProgramImage struct {
	Width      json.Number   `json:"width,omitempty"`
	Height     json.Number   `json:"height,omitempty"`
	URI        string        `json:"uri,omitempty"`
	Ratio      string        `json:"ratio,omitempty"`
	Aspect     string        `json:"aspect,omitempty"`
	Category   string        `json:"category,omitempty"`
	Tier       string        `json:"tier,omitempty"`
	Primary    string        `json:"primary,omitempty"`
	RootID     string        `json:"rootId,omitempty"`
	LastUpdate string        `json:"lastUpdate,omitempty"`
	Caption    *ImageCaption `json:"caption,omitempty"`
}

// ImageCaption is the optional caption block on a ProgramImage.
type ImageCaption struct {
	Content  string `json:"content,omitempty"`
	Language string `json:"lang,omitempty"`
}

// ProgramImageBatchEntry is one element in the POST /metadata/programs/
// response array — programID + image data, or per-program failure with
// code/response inline.
type ProgramImageBatchEntry struct {
	ProgramID string         `json:"programID"`
	Data      []ProgramImage `json:"data,omitempty"`

	// Per-program failure fields.
	Code     int    `json:"code,omitempty"`
	Response string `json:"response,omitempty"`
	Message  string `json:"message,omitempty"`
}

// GetProgramImages calls GET /metadata/programs/{programID} and returns the
// image metadata array for one program (or rootId).
//
// Send the full 14-character programID for episode/movie/sports artwork; the
// 10-character form is deprecated and returns only series-level artwork.
//
// Token-required.
func (c *Client) GetProgramImages(ctx context.Context, programID string) ([]ProgramImage, error) {
	u := c.BaseURL + "metadata/programs/" + url.PathEscape(programID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out []ProgramImage
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetProgramImagesBatch calls POST /metadata/programs/ with up to 500
// programIDs and returns per-program image metadata.
//
// Each ProgramImageBatchEntry may carry per-program failure (Code != 0);
// consumers iterate and dispatch on Code.
//
// Token-required.
func (c *Client) GetProgramImagesBatch(ctx context.Context, programIDs []string) ([]ProgramImageBatchEntry, error) {
	body, err := json.Marshal(programIDs)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal programIDs: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"metadata/programs/", bytes.NewReader(body))
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
	var out []ProgramImageBatchEntry
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GenericDescription is the response value for one entry in POST
// /metadata/description/.
//
// SD returns generic series-level descriptions keyed by SH-base programID
// (the leftmost 10 characters of the EP-prefix programIDs sent in the request).
type GenericDescription struct {
	Code            int    `json:"code,omitempty"`
	Description100  string `json:"description100,omitempty"`
	Description1000 string `json:"description1000,omitempty"`
}

// GetProgramDescriptions calls POST /metadata/description/ with up to 500
// EP-prefix programIDs and returns generic series descriptions keyed by
// SH-base programID.
//
// Token-required.
func (c *Client) GetProgramDescriptions(ctx context.Context, programIDs []string) (map[string]GenericDescription, error) {
	body, err := json.Marshal(programIDs)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal programIDs: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"metadata/description/", bytes.NewReader(body))
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
	var out map[string]GenericDescription
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SportsStatusResponse is the response from GET /metadata/stillRunning/{programId}.
//
// IMPORTANT: empirical testing has shown SD returns HTTP 404 text/html for
// valid sports event programIDs during live games (observed 2026-04-20).
// Endpoint behavior may have drifted from documentation. Consumers should
// validate empirically before relying on this endpoint and be tolerant of
// non-JSON responses.
type SportsStatusResponse struct {
	BaseResponse
	ProgramID    string                  `json:"programID,omitempty"`
	IsRunning    bool                    `json:"isRunning,omitempty"`
	Result       *SportsStatusResult     `json:"result,omitempty"`
	EventDetails *SportsStatusEvent      `json:"eventDetails,omitempty"`
}

// SportsStatusResult carries score/status info; shape varies by sport.
// Defined as a generic key/value map because the wire is sport-specific.
type SportsStatusResult struct {
	HomeTeam string          `json:"homeTeam,omitempty"`
	AwayTeam string          `json:"awayTeam,omitempty"`
	Score    json.RawMessage `json:"score,omitempty"`
	Status   string          `json:"status,omitempty"`
}

// SportsStatusEvent is the optional event-details block.
type SportsStatusEvent struct {
	StartDateTime string `json:"startDateTime,omitempty"`
	Venue         string `json:"venue,omitempty"`
}

// GetSportsEventStatus calls GET /metadata/stillRunning/{programID} and
// returns live-status / score for an in-progress sports event.
//
// Documented codes: 0 OK (in progress or complete), 6001 PROGRAMID_QUEUED
// (retry in 30s), 6002 FUTURE_PROGRAM, 6000 INVALID_PROGRAMID. See
// SportsStatusResponse doc-comment for empirical behavior caveat.
//
// Token-required.
func (c *Client) GetSportsEventStatus(ctx context.Context, programID string) (*SportsStatusResponse, error) {
	u := c.BaseURL + "metadata/stillRunning/" + url.PathEscape(programID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out SportsStatusResponse
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
