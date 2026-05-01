package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Version is the response from GET /version/{clientName}.
type Version struct {
	BaseResponse
	Client  string `json:"client"`
	Version string `json:"version"`
}

// GetVersion returns the latest registered version of the named client.
//
// No authentication is required. SD returns UNKNOWN_CLIENT (1005) for a name
// it does not recognize.
func (c *Client) GetVersion(ctx context.Context, clientName string) (*Version, error) {
	u := c.BaseURL + "version/" + url.PathEscape(clientName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var v Version
	if err := c.readResponse(resp, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
