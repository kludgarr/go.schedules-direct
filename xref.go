package schedulesdirect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// LanguageCrossReference is one entry in the per-programID array returned by
// POST /xref. Each entry describes an alternate-language version of the
// requested program.
//
// Wire shape includes both descriptionLanguage / descriptionLanguageName
// (locale + display name) and titleLanguage / titleLanguageName, plus the
// usual hash + md5 change-detection fields.
type LanguageCrossReference struct {
	ProgramID               string `json:"programID,omitempty"`
	DescriptionLanguage     string `json:"descriptionLanguage,omitempty"`
	DescriptionLanguageName string `json:"descriptionLanguageName,omitempty"`
	TitleLanguage           string `json:"titleLanguage,omitempty"`
	TitleLanguageName       string `json:"titleLanguageName,omitempty"`
	Hash                    string `json:"hash,omitempty"`
	MD5                     string `json:"md5,omitempty"`
}

// GetLanguageCrossReferences calls POST /xref with up to 500 programIDs and
// returns alternate-language program references keyed by source programID.
//
// Token-required.
func (c *Client) GetLanguageCrossReferences(ctx context.Context, programIDs []string) (map[string][]LanguageCrossReference, error) {
	body, err := json.Marshal(programIDs)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal programIDs: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"xref", bytes.NewReader(body))
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
	var out map[string][]LanguageCrossReference
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
