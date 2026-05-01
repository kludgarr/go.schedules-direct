package schedulesdirect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Program is one entry in the POST /programs response array.
//
// Two shapes share this struct:
//   - Success: typed fields populated, Code is 0.
//   - Per-program failure: Code != 0 (typically 6000 INVALID_PROGRAMID,
//     6001 PROGRAMID_QUEUED, 6002 FUTURE_PROGRAM); auxiliary fields like
//     RetryTime / EventStartDateTime may be populated. Typed fields may be
//     empty in this case.
//
// Consumers iterate the response array and check Code per entry; per-program
// failures do NOT cause the top-level call to return an *Error.
//
// HasImageArtwork is deprecated by SD; consumers should prefer the more
// specific HasSeriesArtwork / HasEpisodeArtwork / HasMovieArtwork /
// HasSportsArtwork booleans.
type Program struct {
	ProgramID       string `json:"programID"`
	ResourceID      string `json:"resourceID,omitempty"`
	MD5             string `json:"md5,omitempty"`
	Hash            string `json:"hash,omitempty"`

	Titles           []ProgramTitle      `json:"titles,omitempty"`
	EpisodeTitle150  string              `json:"episodeTitle150,omitempty"`
	Descriptions     ProgramDescriptions `json:"descriptions,omitempty"`
	OriginalAirDate  string              `json:"originalAirDate,omitempty"`
	ShowType         string              `json:"showType,omitempty"`
	EntityType       string              `json:"entityType,omitempty"`
	Country          []string            `json:"country,omitempty"`
	Genres           []string            `json:"genres,omitempty"`
	Cast             []CastMember        `json:"cast,omitempty"`
	Crew             []CrewMember        `json:"crew,omitempty"`
	ContentAdvisory  []string            `json:"contentAdvisory,omitempty"`
	ContentRating    []ContentRating     `json:"contentRating,omitempty"`
	KeyWords         json.RawMessage     `json:"keyWords,omitempty"`
	Metadata         []ProgramMetadata   `json:"metadata,omitempty"`
	Duration         int                 `json:"duration,omitempty"`
	OfficialURL      string              `json:"officialURL,omitempty"`
	EpisodeImage     string              `json:"episodeImage,omitempty"`

	// Sport-event-only / movie-only nested structures. Defined as RawMessage
	// to keep the core type surface tractable; consumers that need them call
	// json.Unmarshal on the field directly.
	EventDetails    json.RawMessage `json:"eventDetails,omitempty"`
	Movie           json.RawMessage `json:"movie,omitempty"`
	Recommendations json.RawMessage `json:"recommendations,omitempty"`
	Awards          json.RawMessage `json:"awards,omitempty"`
	MultiPart       *Multipart      `json:"multiPart,omitempty"`

	// Artwork-availability flags. HasImageArtwork is deprecated.
	HasSeriesArtwork  bool `json:"hasSeriesArtwork,omitempty"`
	HasEpisodeArtwork bool `json:"hasEpisodeArtwork,omitempty"`
	HasSeasonArtwork  bool `json:"hasSeasonArtwork,omitempty"`
	HasMovieArtwork   bool `json:"hasMovieArtwork,omitempty"`
	HasSportsArtwork  bool `json:"hasSportsArtwork,omitempty"`
	HasImageArtwork   bool `json:"hasImageArtwork,omitempty"` // deprecated by SD

	// Per-program failure fields. Populated when Code != 0.
	Code               int    `json:"code,omitempty"`
	Response           string `json:"response,omitempty"`
	Message            string `json:"message,omitempty"`
	RetryTime          string `json:"retryTime,omitempty"`
	EventStartDateTime string `json:"eventStartDateTime,omitempty"`
}

// ProgramTitle is one entry in Program.Titles. title120 is mandatory;
// titleLanguage will become mandatory in a future API version.
type ProgramTitle struct {
	Title120      string `json:"title120"`
	TitleLanguage string `json:"titleLanguage,omitempty"`
}

// ProgramDescriptions is the descriptions block on Program.
//
// SD provides up to two length-bucketed description sets per program; consumers
// typically prefer Description1000 over Description100 for richer rendering.
type ProgramDescriptions struct {
	Description100  []DescriptionEntry `json:"description100,omitempty"`
	Description1000 []DescriptionEntry `json:"description1000,omitempty"`
}

// DescriptionEntry is one localized description string.
type DescriptionEntry struct {
	DescriptionLanguage string `json:"descriptionLanguage"`
	Description         string `json:"description"`
}

// CastMember is one entry in Program.Cast.
//
// NameID and PersonID are stringified numeric IDs — do not convert to integer
// (SD reserves the option to switch to non-numeric forms).
type CastMember struct {
	BillingOrder  string `json:"billingOrder"`
	Role          string `json:"role"`
	Name          string `json:"name"`
	CharacterName string `json:"characterName,omitempty"`
	NameID        string `json:"nameId,omitempty"`
	PersonID      string `json:"personId,omitempty"`
}

// CrewMember is one entry in Program.Crew. Same shape as CastMember without
// CharacterName.
type CrewMember struct {
	BillingOrder string `json:"billingOrder"`
	Role         string `json:"role"`
	Name         string `json:"name"`
	NameID       string `json:"nameId,omitempty"`
	PersonID     string `json:"personId,omitempty"`
}

// ContentRating is one entry in Program.ContentRating.
type ContentRating struct {
	Body           string   `json:"body,omitempty"`
	Code           string   `json:"code,omitempty"`
	Country        string   `json:"country,omitempty"`
	ContentWarning []string `json:"contentWarning,omitempty"`
}

// ProgramMetadata is one entry in Program.Metadata. Each entry typically
// contains one of Gracenote / TVmaze; both may be present on the same entry.
type ProgramMetadata struct {
	Gracenote *GracenoteMetadata `json:"Gracenote,omitempty"`
	TVmaze    *TVmazeMetadata    `json:"TVmaze,omitempty"`
}

// GracenoteMetadata is the Gracenote-sourced episode/season block on
// ProgramMetadata.
//
// TotalEpisodes meaning depends on programID prefix: for EP it is total
// episodes in this season; for SH it is total episodes in the series.
type GracenoteMetadata struct {
	Season         int    `json:"season,omitempty"`
	Episode        int    `json:"episode,omitempty"`
	TotalEpisodes  int    `json:"totalEpisodes,omitempty"`
	TotalSeasons   int    `json:"totalSeasons,omitempty"`
	ProgramVersion string `json:"programVersion,omitempty"`
}

// TVmazeMetadata is the TVmaze-sourced episode/season block on
// ProgramMetadata.
type TVmazeMetadata struct {
	Season        int    `json:"season,omitempty"`
	Episode       int    `json:"episode,omitempty"`
	TotalEpisodes int    `json:"totalEpisodes,omitempty"`
	URL           string `json:"url,omitempty"`
}

// GetPrograms calls POST /programs with the requested programIDs and returns
// per-program details.
//
// Each Program may carry per-program failure (Code != 0); consumers iterate
// and dispatch on Code (6000 INVALID_PROGRAMID is permanent; 6001
// PROGRAMID_QUEUED and 6002 FUTURE_PROGRAM are soft).
//
// Max 5000 programIDs per call (SD-imposed). Token-required.
func (c *Client) GetPrograms(ctx context.Context, programIDs []string) ([]Program, error) {
	body, err := json.Marshal(programIDs)
	if err != nil {
		return nil, fmt.Errorf("schedulesdirect: marshal programIDs: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"programs", bytes.NewReader(body))
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
	var out []Program
	if err := c.readResponse(resp, &out); err != nil {
		return nil, err
	}
	return out, nil
}
