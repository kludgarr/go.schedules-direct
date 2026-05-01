package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// DeleteMessage calls DELETE /messages/{messageID}.
//
// Deprecated: SD documents this endpoint but reference clients have not
// implemented it. Messages in /status notifications[] and account.messages[]
// are plain strings with no embedded ID, so the messageID parameter has no
// observable source in practice. Provided for spec completeness; consumers
// should not rely on it.
//
// Token-required.
func (c *Client) DeleteMessage(ctx context.Context, messageID string) (*BaseResponse, error) {
	u := c.BaseURL + "messages/" + url.PathEscape(messageID)
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
	var out BaseResponse
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
