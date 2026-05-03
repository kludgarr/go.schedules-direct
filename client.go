package schedulesdirect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrMissingUserAgent is returned by NewClient when called with an empty
// userAgent. SD rejects requests without a User-Agent header (error code 1003);
// the library does not synthesize a default. The consumer must provide one.
var ErrMissingUserAgent = errors.New("schedulesdirect: user agent is required (SD rejects requests without one; error code 1003)")

// Client is a Schedules Direct REST API client.
//
// Construct with NewClient. HTTPClient is consumer-configurable via WithHTTPClient;
// any cross-cutting concerns (rate limiting, single-flight, observability) are
// outside this library's scope and are the consumer's call as to where and how
// to implement.
type Client struct {
	// HTTPClient performs the underlying HTTP requests.
	// Defaults to http.DefaultClient.
	HTTPClient *http.Client

	// BaseURL is the SD API base URL, including trailing slash and version
	// segment. Defaults to BaseURL constant.
	BaseURL string

	// UserAgent is the User-Agent header sent on every request. Required;
	// passed through verbatim from the value supplied to NewClient. The library
	// does not synthesize a default — SD rejects requests without one (error
	// code 1003), and which UA to send is a consumer concern.
	UserAgent string

	// Token is the SD-issued bearer token. Required on every endpoint except
	// /token itself. Consumers manage acquisition, expiry, and refresh.
	Token string
}

// ClientOption configures a Client at construction time.
type ClientOption func(*Client)

// NewClient constructs a Client. userAgent is required and is passed through
// verbatim on every request; an empty string returns ErrMissingUserAgent.
func NewClient(userAgent string, opts ...ClientOption) (*Client, error) {
	if userAgent == "" {
		return nil, ErrMissingUserAgent
	}
	c := &Client{
		HTTPClient: http.DefaultClient,
		BaseURL:    BaseURL,
		UserAgent:  userAgent,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// WithHTTPClient overrides the default *http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) { c.HTTPClient = hc }
}

// WithBaseURL overrides the default BaseURL. Useful for testing against an
// httptest.Server. Must end in a trailing slash.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) { c.BaseURL = url }
}

// WithToken sets the bearer token used on token-required endpoints.
func WithToken(tok string) ClientOption {
	return func(c *Client) { c.Token = tok }
}

// readResponse consumes resp.Body and unmarshals the JSON into into.
//
// SD's wire shape varies by endpoint: most return an object with the standard
// BaseResponse envelope (code, response, message, ...); some (e.g. /available,
// /available/dvb-s, /headends) return arrays at top level with no envelope.
// Errors are uniformly object-shaped per ErrorResponse.
//
// Dispatch:
//   - If the body's first non-whitespace byte is '[', the response is an array;
//     unmarshal directly into into (no envelope check).
//   - If '{', unmarshal into BaseResponse; if Code != 0, parse the body as
//     *Error and return it. Otherwise unmarshal into into.
//
// Pass into=nil to discard the success body after envelope check.
func (c *Client) readResponse(resp *http.Response, into any) error {
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("schedulesdirect: read body: %w", err)
	}

	switch firstJSONByte(raw) {
	case '[':
		if into != nil {
			if err := json.Unmarshal(raw, into); err != nil {
				return fmt.Errorf("schedulesdirect: parse response: %w", err)
			}
		}
		return nil
	case '{':
		var base BaseResponse
		if err := json.Unmarshal(raw, &base); err != nil {
			return fmt.Errorf("schedulesdirect: parse envelope: %w", err)
		}
		if base.Code != 0 {
			var sdErr Error
			if err := json.Unmarshal(raw, &sdErr); err != nil {
				return fmt.Errorf("schedulesdirect: parse error envelope: %w", err)
			}
			return &sdErr
		}
		if into != nil {
			if err := json.Unmarshal(raw, into); err != nil {
				return fmt.Errorf("schedulesdirect: parse response: %w", err)
			}
		}
		return nil
	default:
		return fmt.Errorf("schedulesdirect: unexpected response shape (first byte %q): %s", firstJSONByte(raw), truncate(raw, 200))
	}
}

func firstJSONByte(b []byte) byte {
	for _, c := range b {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return c
		}
	}
	return 0
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
