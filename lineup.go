package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// UserLineupsResponse is the response from GET /lineups — the lineups
// currently added to the account.
type UserLineupsResponse struct {
	BaseResponse
	Lineups []UserLineup `json:"lineups"`
}

// UserLineup is one entry in UserLineupsResponse.Lineups.
type UserLineup struct {
	Lineup    string `json:"lineup"`
	Name      string `json:"name,omitempty"`
	Transport string `json:"transport,omitempty"`
	Location  string `json:"location,omitempty"`
	URI       string `json:"uri,omitempty"`
	IsDeleted bool   `json:"isDeleted,omitempty"`
}

// LineupMapping is the response from GET /lineups/{lineupID} — full
// channel-to-station mapping, station details, and metadata.
//
// Note: SD wire does NOT envelope this response with code/response/serverID
// fields (synthesis gap; the spec includes BaseResponse via $ref). Errors
// are still object-shaped and envelope-checked correctly by the client.
type LineupMapping struct {
	Map      []ChannelMap    `json:"map,omitempty"`
	Stations []Station       `json:"stations,omitempty"`
	Metadata LineupMetadata  `json:"metadata,omitempty"`
}

// ChannelMap is one entry in LineupMapping.Map.
//
// Fields populated depend on transport type. ATSC OTA: ATSCMajor, ATSCMinor,
// UHFVHF, ATSCType. DVB: NetworkID, ServiceID, TransportID, FrequencyHz,
// Polarization, ModulationSystem, DeliverySystem, FEC, Symbolrate. QAM:
// VirtualChannel, ChannelMajor, ChannelMinor. Verbose-map opt-in: ProviderCallsign,
// LogicalChannelNumber, MatchType.
type ChannelMap struct {
	StationID            string `json:"stationID"`
	Channel              string `json:"channel"`
	UHFVHF               int    `json:"uhfVhf,omitempty"`
	ATSCMajor            int    `json:"atscMajor,omitempty"`
	ATSCMinor            int    `json:"atscMinor,omitempty"`
	ATSCType             string `json:"atscType,omitempty"`
	FrequencyHz          int    `json:"frequencyHz,omitempty"`
	ServiceID            int    `json:"serviceID,omitempty"`
	NetworkID            int    `json:"networkID,omitempty"`
	TransportID          int    `json:"transportID,omitempty"`
	Polarization         string `json:"polarization,omitempty"`
	DeliverySystem       string `json:"deliverySystem,omitempty"`
	ModulationSystem     string `json:"modulationSystem,omitempty"`
	Symbolrate           int    `json:"symbolrate,omitempty"`
	FEC                  string `json:"fec,omitempty"`
	VirtualChannel       string `json:"virtualChannel,omitempty"`
	ChannelMajor         int    `json:"channelMajor,omitempty"`
	ChannelMinor         int    `json:"channelMinor,omitempty"`
	ProviderCallsign     string `json:"providerCallsign,omitempty"`
	LogicalChannelNumber string `json:"logicalChannelNumber,omitempty"`
	MatchType            string `json:"matchType,omitempty"`
}

// Station is one entry in LineupMapping.Stations.
type Station struct {
	StationID           string       `json:"stationID"`
	Name                string       `json:"name,omitempty"`
	Callsign            string       `json:"callsign,omitempty"`
	Affiliate           string       `json:"affiliate,omitempty"`
	BroadcastLanguage   []string     `json:"broadcastLanguage,omitempty"`
	DescriptionLanguage []string     `json:"descriptionLanguage,omitempty"`
	Broadcaster         *Broadcaster `json:"broadcaster,omitempty"`
	IsCommercialFree    bool         `json:"isCommercialFree,omitempty"`
	StationLogo         []StationLogo `json:"stationLogo,omitempty"`
	Logo                *StationLogo  `json:"logo,omitempty"` // deprecated singular form; consumer prefers StationLogo
	URL                 string       `json:"URL,omitempty"`
	IsRadioStation      bool         `json:"isRadioStation,omitempty"`
}

// Broadcaster is the geographic context of a Station.
type Broadcaster struct {
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postalcode,omitempty"`
	Country    string `json:"country,omitempty"`
}

// StationLogo is one logo variant within a Station.
//
// Hash is the same value as MD5 — SD's wire returns both fields with
// identical content (synthesis gap; OpenAPI spec lists only md5).
type StationLogo struct {
	URL      string `json:"URL,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	MD5      string `json:"md5,omitempty"`
	Hash     string `json:"hash,omitempty"`
	Source   string `json:"source,omitempty"`
	Category string `json:"category,omitempty"`
}

// LineupMetadata is the per-lineup metadata block in LineupMapping.
type LineupMetadata struct {
	Lineup     string `json:"lineup"`
	Modified   string `json:"modified,omitempty"`
	Transport  string `json:"transport,omitempty"`
	Modulation string `json:"modulation,omitempty"`
}

// LineupChangeResponse is the response from PUT or DELETE /lineups/{lineupID}.
//
// ChangesRemaining is the number of lineup add/delete operations remaining in
// the current 24-hour window. Wire is consistently an integer; the OpenAPI
// spec's oneOf int/string was over-cautious — wire confirms int.
type LineupChangeResponse struct {
	BaseResponse
	ChangesRemaining int `json:"changesRemaining,omitempty"`
}

// LineupPreviewChannel is one entry in the GET /lineups/preview/{lineupID}
// response array.
type LineupPreviewChannel struct {
	Channel   string `json:"channel,omitempty"`
	Name      string `json:"name,omitempty"`
	Callsign  string `json:"callsign,omitempty"`
	Affiliate string `json:"affiliate,omitempty"`
}

// GetLineups calls GET /lineups and returns the lineups currently added to
// the account. Token-required.
func (c *Client) GetLineups(ctx context.Context) (*UserLineupsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"lineups", nil)
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
	var out UserLineupsResponse
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetLineupMapping calls GET /lineups/{lineupID} and returns the full
// channel-to-stationID mapping plus station details and metadata.
//
// When verbose is true, the verboseMap header is sent and returned ChannelMap
// entries include ProviderCallsign / LogicalChannelNumber / MatchType.
//
// Token-required.
func (c *Client) GetLineupMapping(ctx context.Context, lineupID string, verbose bool) (*LineupMapping, error) {
	u := c.BaseURL + "lineups/" + url.PathEscape(lineupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}
	if verbose {
		req.Header.Set("verboseMap", "true")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out LineupMapping
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddLineup calls PUT /lineups/{lineupID} to add a lineup to the account.
//
// SD limits this to 6 add operations per 24-hour period (combined with
// DeleteLineup); the response carries ChangesRemaining. Token-required.
func (c *Client) AddLineup(ctx context.Context, lineupID string) (*LineupChangeResponse, error) {
	u := c.BaseURL + "lineups/" + url.PathEscape(lineupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, nil)
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
	var out LineupChangeResponse
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteLineup calls DELETE /lineups/{lineupID} to remove a lineup from the
// account. Counts against the same 24h add/delete budget as AddLineup.
// Token-required.
func (c *Client) DeleteLineup(ctx context.Context, lineupID string) (*LineupChangeResponse, error) {
	u := c.BaseURL + "lineups/" + url.PathEscape(lineupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
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
	var out LineupChangeResponse
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PreviewLineup calls GET /lineups/preview/{lineupID} and returns a channel
// preview without adding the lineup. Token-required.
func (c *Client) PreviewLineup(ctx context.Context, lineupID string) ([]LineupPreviewChannel, error) {
	u := c.BaseURL + "lineups/preview/" + url.PathEscape(lineupID)
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
	var out []LineupPreviewChannel
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
