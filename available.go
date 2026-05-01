package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// AvailableService is one entry in the GET /available response — a service
// type SD offers (countries, languages, DVB-T transmitters, DVB-S
// satellites) with the URI to fetch its data.
type AvailableService struct {
	Description string `json:"description"`
	Type        string `json:"type"`
	URI         string `json:"uri"`
}

// Country is one country entry within a region in GET /available/countries.
//
// PostalCode is a regex pattern (with leading/trailing slashes in some
// entries) that headend-lookup queries must match. PostalCodeExample shows
// a sample input. OnePostalCode=true means the country has a single valid
// postal code (commonly small territories).
type Country struct {
	FullName          string `json:"fullName"`
	ShortName         string `json:"shortName"`
	PostalCode        string `json:"postalCode,omitempty"`
	PostalCodeExample string `json:"postalCodeExample,omitempty"`
	OnePostalCode     bool   `json:"onePostalCode,omitempty"`
}

// GetAvailableServices calls GET /available. No authentication required.
func (c *Client) GetAvailableServices(ctx context.Context) ([]AvailableService, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"available", nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out []AvailableService
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAvailableCountries calls GET /available/countries. Returns countries
// grouped by region — the outer map key is the region name (e.g. "Caribbean",
// "Europe", "North America"). No authentication required.
func (c *Client) GetAvailableCountries(ctx context.Context) (map[string][]Country, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"available/countries", nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out map[string][]Country
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAvailableLanguages calls GET /available/languages. Returns a map of
// language code (typically ISO 639-1 two-letter or ISO 639-2 three-letter
// code) to language name in English. No authentication required.
func (c *Client) GetAvailableLanguages(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"available/languages", nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out map[string]string
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAvailableDVBS calls GET /available/dvb-s and returns the list of DVB-S
// satellites available to the account. Returns an empty slice if the account
// has no DVB-S coverage in its registered region — currently SD provides
// DVB-S coverage only to accounts with a UK-registered address.
//
// No authentication required.
func (c *Client) GetAvailableDVBS(ctx context.Context) ([]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"available/dvb-s", nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out []any
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetTransmitters calls GET /transmitters/{iso3166alpha3} and returns the
// DVB-T (Freeview) transmitter map for the given country — keys are
// transmitter names, values are lineup IDs. iso3166alpha3 must be a 3-letter
// country code (e.g. "GBR").
//
// SD currently supports only "GBR". Other country codes return code 2050
// INVALID_PARAMETER:COUNTRY as an *Error. No authentication required.
func (c *Client) GetTransmitters(ctx context.Context, iso3166alpha3 string) (map[string]string, error) {
	u := c.BaseURL + "transmitters/" + url.PathEscape(iso3166alpha3)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var out map[string]string
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
