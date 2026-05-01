package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Headend is one entry in the GET /headends response array.
//
// A headend may carry multiple lineups (e.g. a Cable headend with DEFAULT
// analog + X digital + QAM variants). Consumers select a lineup by ID and
// add it via AddLineup.
type Headend struct {
	Headend   string          `json:"headend"`
	Transport string          `json:"transport,omitempty"`
	Location  string          `json:"location,omitempty"`
	Lineups   []HeadendLineup `json:"lineups,omitempty"`
}

// HeadendLineup is one lineup entry within a Headend.
type HeadendLineup struct {
	Lineup string `json:"lineup"`
	Name   string `json:"name,omitempty"`
	URI    string `json:"uri,omitempty"`
}

// GetHeadends calls GET /headends?country={iso3166alpha3}&postalcode={pc} and
// returns the headends available for that country/postal-code combination.
//
// country must be a 3-letter ISO 3166-1 alpha-3 code; postalCode must be valid
// for that country (see GET /available/countries for the per-country pattern).
//
// Token-required.
func (c *Client) GetHeadends(ctx context.Context, country, postalCode string) ([]Headend, error) {
	q := url.Values{}
	q.Set("country", country)
	q.Set("postalcode", postalCode)
	u := c.BaseURL + "headends?" + q.Encode()

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
	var out []Headend
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
