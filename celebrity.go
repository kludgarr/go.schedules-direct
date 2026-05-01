package schedulesdirect

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Celebrity is the response from GET /celebrity/{personId} — biographical
// details for a person.
//
// PersonID, NameID, and similar identifier strings are stringified numeric
// IDs — do not convert to integer (SD reserves the option to switch forms).
type Celebrity struct {
	PersonID    string             `json:"personId"`
	Type        string             `json:"type,omitempty"`
	Gender      string             `json:"gender,omitempty"`
	BirthDate   string             `json:"birthDate,omitempty"`
	BirthPlace  string             `json:"birthPlace,omitempty"`
	UpdateDate  string             `json:"updateDate,omitempty"`
	Names       []CelebrityName    `json:"names,omitempty"`
	Images      []CelebrityImage   `json:"images,omitempty"`
	Mediography []MediographyEntry `json:"mediography,omitempty"`
	Awards      []CelebrityAward   `json:"awards,omitempty"`
}

// CelebrityName is one entry in Celebrity.Names. A person may have multiple
// nameIds (marriage, alternate spellings) but a single personId.
type CelebrityName struct {
	NameID    string `json:"nameId"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
}

// CelebrityImage is one image-metadata entry returned from
// /metadata/celebrity/{personId} or embedded in Celebrity.Images.
//
// Width/Height are json.Number to accept both integer and stringified-integer
// forms (legacy wire returns strings; consistent with the ProgramImage
// width/height oneOf shape).
type CelebrityImage struct {
	URI         string      `json:"uri,omitempty"`
	Width       json.Number `json:"width,omitempty"`
	Height      json.Number `json:"height,omitempty"`
	Aspect      string      `json:"aspect,omitempty"`
	Ratio       string      `json:"ratio,omitempty"`
	Category    string      `json:"category,omitempty"`
	Tier        string      `json:"tier,omitempty"`
	LastUpdated string      `json:"lastUpdated,omitempty"`
}

// MediographyEntry is one program credit in Celebrity.Mediography.
type MediographyEntry struct {
	ProgramID            string             `json:"programID"`
	Title                string             `json:"title,omitempty"`
	ProgramType          string             `json:"programType,omitempty"`
	ProgramTitleLanguage string             `json:"programTitleLanguage,omitempty"`
	Year                 string             `json:"year,omitempty"`
	Credits              []MediographyCredit `json:"credits,omitempty"`
}

// MediographyCredit is one role within a MediographyEntry.
type MediographyCredit struct {
	Role          string `json:"role"`
	CharacterName string `json:"characterName,omitempty"`
}

// CelebrityAward is one entry in Celebrity.Awards.
type CelebrityAward struct {
	AwardName string `json:"awardName,omitempty"`
	Category  string `json:"category,omitempty"`
	Year      string `json:"year,omitempty"`
	Won       bool   `json:"won,omitempty"`
	ProgramID string `json:"programID,omitempty"`
}

// GetCelebrity calls GET /celebrity/{personId} and returns a Celebrity record.
// PersonID must be the stringified numeric ID. Token-required.
func (c *Client) GetCelebrity(ctx context.Context, personID string) (*Celebrity, error) {
	u := c.BaseURL + "celebrity/" + url.PathEscape(personID)
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
	var out Celebrity
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCelebrityImages calls GET /metadata/celebrity/{personId} and returns the
// celebrity's image metadata.
//
// Note: SD wiki indicates this endpoint may not require a token, while the
// main API page indicates it does. The library sends the token if available;
// behavior may vary.
func (c *Client) GetCelebrityImages(ctx context.Context, personID string) ([]CelebrityImage, error) {
	u := c.BaseURL + "metadata/celebrity/" + url.PathEscape(personID)
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
	var out []CelebrityImage
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
