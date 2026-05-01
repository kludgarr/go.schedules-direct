package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
)

// Status is the response from GET /status — account state, subscribed
// lineups, last data update, system-wide notifications, and per-component
// system status.
type Status struct {
	BaseResponse
	Account        AccountStatus  `json:"account"`
	Lineups        []StatusLineup `json:"lineups"`
	LastDataUpdate string         `json:"lastDataUpdate,omitempty"`
	Notifications  []string       `json:"notifications,omitempty"`
	SystemStatus   []SystemStatus `json:"systemStatus,omitempty"`
	TokenExpires   int64          `json:"tokenExpires,omitempty"`
	ServerTime     int64          `json:"serverTime,omitempty"`
}

// AccountStatus is the per-account block within Status.
//
// AccountExpiration (UNIX epoch) is the canonical numeric form; Expires is
// the ISO-8601 string form of the same value. MaxLineups is the SD-imposed
// cap (currently 4 per default plan; see code 4101 MAX_LINEUPS). Messages
// carries per-account SD announcements as plain strings.
type AccountStatus struct {
	Expires           string   `json:"expires,omitempty"`
	AccountExpiration int64    `json:"accountExpiration,omitempty"`
	Messages          []string `json:"messages,omitempty"`
	MaxLineups        int      `json:"maxLineups,omitempty"`
}

// StatusLineup is a per-lineup entry in Status.Lineups.
//
// For active lineups, the SD ID lives in Lineup; for entries with
// IsDeleted=true, SD's wire uses the alternate ID field instead of Lineup
// (a documented wire inconsistency not reflected in the OpenAPI synthesis).
// Consumers that need the ID regardless of deletion state should consult
// Lineup if non-empty, otherwise ID. URI is the relative path to the lineup
// detail (GET /lineups/{lineupID}).
type StatusLineup struct {
	Lineup    string `json:"lineup,omitempty"`
	ID        string `json:"ID,omitempty"`
	Modified  string `json:"modified,omitempty"`
	URI       string `json:"uri,omitempty"`
	Name      string `json:"name,omitempty"`
	IsDeleted bool   `json:"isDeleted,omitempty"`
}

// SystemStatus is a per-component status entry in Status.SystemStatus.
//
// Status is typically "Online" / "Offline" — consumers map to whatever
// service-health classification they use (see canonical SD spec context for
// the operator-derived service-status classification).
type SystemStatus struct {
	Date    string `json:"date,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GetStatus calls GET /status and returns the account + system status.
//
// Requires a token. If c.Token is empty, the library sends the request
// without a token header and SD returns TOKEN_MISSING (code 1004) as an
// *Error.
func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"status", nil)
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
	var s Status
	if err := c.readResponse(resp, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
