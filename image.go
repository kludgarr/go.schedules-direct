package schedulesdirect

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GetImageURL calls GET /image/{uri} and returns the SD-issued S3 URL the
// image lives at. SD responds with HTTP 303 redirect; the library reads the
// Location header rather than following the redirect, so the consumer can
// fetch the image bytes on their own schedule (with their own caching,
// gzip handling, and rate-limiting).
//
// The returned URL is valid for 120 seconds — fetch promptly. Do not cache
// the URL itself; cache the resulting bytes (keyed by content hash) instead.
//
// Image-quota awareness: SD imposes a per-account daily image-download limit.
// Exceeding it returns code 5002 (or 5003 for trial). Consumers MUST stop
// requesting images on 5002/5003 to avoid an account block.
//
// Note: the image URI returned in metadata responses may already be a
// fully-qualified S3 URL (containing "https://"); in that case, fetch it
// directly without calling this endpoint.
//
// Token-required.
func (c *Client) GetImageURL(ctx context.Context, uri string) (string, error) {
	u := c.BaseURL + "image/" + url.PathEscape(uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	// Non-redirect-following copy of the underlying client; shares Transport
	// so any consumer-installed Transport customization still applies.
	nonRedirect := *c.HTTPClient
	nonRedirect.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := nonRedirect.Do(req)
	if err != nil {
		return "", fmt.Errorf("schedulesdirect: http: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusSeeOther, http.StatusFound, http.StatusMovedPermanently, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		loc := resp.Header.Get("Location")
		if loc == "" {
			return "", fmt.Errorf("schedulesdirect: %d response missing Location header", resp.StatusCode)
		}
		return loc, nil
	default:
		// Body should carry an ErrorResponse envelope.
		raw, _ := io.ReadAll(resp.Body)
		var sdErr Error
		if json.Unmarshal(raw, &sdErr) == nil && sdErr.Code != 0 {
			return "", &sdErr
		}
		return "", fmt.Errorf("schedulesdirect: unexpected status %d (body: %s)", resp.StatusCode, truncate(raw, 200))
	}
}
