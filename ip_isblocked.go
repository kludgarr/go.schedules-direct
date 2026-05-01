package schedulesdirect

import (
	"context"
	"fmt"
	"net/http"
)

// IPBlockStatus is the response from GET /ip_isblocked.
//
// BlockedOnLoadBalancer distinguishes service-state from load-balancer-state
// per the documented service-vs-account-status separation: a true value
// signals the consumer is being blocked at the LB layer rather than SD itself
// being down.
type IPBlockStatus struct {
	BaseResponse
	BlockedOnLoadBalancer bool `json:"blocked_on_load_balancer"`
}

// GetIPBlockStatus calls GET /ip_isblocked and returns whether the client IP
// is being blocked on SD's load balancer.
//
// No authentication is required. SD rate-limits this endpoint to 100 calls
// per 24h (resets at 00:00Z); enforcement is consumer responsibility.
func (c *Client) GetIPBlockStatus(ctx context.Context) (*IPBlockStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"ip_isblocked", nil)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: http: %w", err)
	}
	var s IPBlockStatus
	if err := c.readResponse(resp, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
