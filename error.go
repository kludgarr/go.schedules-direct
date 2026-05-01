package schedulesdirect

import "fmt"

// BaseResponse is the envelope present on every Schedules Direct response,
// success or failure. Endpoint-specific response types embed BaseResponse.
type BaseResponse struct {
	// Code is the numeric outcome. 0 = success; non-zero = error.
	Code int `json:"code"`

	// Response is SD's internal-code label. "OK" on success;
	// "SERVICE_OFFLINE" / "INVALID_USER" / etc. on failure. Pairs 1:1 with Code.
	Response string `json:"response,omitempty"`

	// Message is SD's human-readable message.
	Message string `json:"message,omitempty"`

	// ServerID identifies the SD edge server that produced the response.
	ServerID string `json:"serverID,omitempty"`

	// Datetime is SD's ISO-8601 timestamp on the response.
	Datetime string `json:"datetime,omitempty"`
}

// Error is a non-zero envelope code returned by Schedules Direct.
//
// SD wraps every response in BaseResponse; when Code != 0, the library returns
// *Error rather than the typed response. Several context fields are populated
// only with specific codes (e.g. RetryTime with SCHEDULE_QUEUED;
// AccountExpiration with ACCOUNT_EXPIRED; MinDate / MaxDate / RequestedDate
// with SCHEDULE_RANGE_EXCEEDED). Consumers use errors.As to unwrap:
//
//	var sdErr *schedulesdirect.Error
//	if errors.As(err, &sdErr) && sdErr.Code == schedulesdirect.CodeServiceOffline {
//		// retry later
//	}
//
// The classification of which codes are "fatal" / "retryable" / "warning" is
// consumer policy and outside this library's scope.
type Error struct {
	BaseResponse

	// ServerTime is SD's clock as UNIX epoch seconds, when present.
	ServerTime int64 `json:"serverTime,omitempty"`

	// Token may be present on SERVICE_OFFLINE (placeholder; not a valid token).
	Token string `json:"token,omitempty"`

	// TokenExpires may be present on SERVICE_OFFLINE (often 0).
	TokenExpires int64 `json:"tokenExpires,omitempty"`

	// RetryTime is populated for SCHEDULE_QUEUED (code 7100): honor before retry.
	RetryTime string `json:"retryTime,omitempty"`

	// AccountExpiration is populated for ACCOUNT_EXPIRED (code 4001): UNIX epoch.
	AccountExpiration int64 `json:"accountExpiration,omitempty"`

	// EventStartDateTime is populated for FUTURE_PROGRAM (code 6002).
	EventStartDateTime string `json:"eventStartDateTime,omitempty"`

	// ProgramID is populated for PROGRAMID_QUEUED / FUTURE_PROGRAM.
	ProgramID string `json:"programID,omitempty"`

	// StationID is populated for SCHEDULE_QUEUED.
	StationID string `json:"stationID,omitempty"`

	// MinDate is populated for SCHEDULE_RANGE_EXCEEDED: earliest available date.
	MinDate string `json:"minDate,omitempty"`

	// MaxDate is populated for SCHEDULE_RANGE_EXCEEDED: latest available date.
	MaxDate string `json:"maxDate,omitempty"`

	// RequestedDate is populated for SCHEDULE_RANGE_EXCEEDED: out-of-range date.
	RequestedDate string `json:"requestedDate,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Response != "" {
		return fmt.Sprintf("schedulesdirect: code %d (%s): %s", e.Code, e.Response, e.Message)
	}
	return fmt.Sprintf("schedulesdirect: code %d: %s", e.Code, e.Message)
}

// SD error code constants. Sourced from the canonical OpenAPI spec
// (schedulesdirect_pseudo_openapi-v3.0.3.json) ErrorResponse.response enum.
// Library exposes the full set so consumers can match against named constants
// rather than magic numbers.
const (
	CodeOK = 0

	// Request validation (1xxx)
	CodeInvalidJSON          = 1001
	CodeUserAgentRequired    = 1003
	CodeTokenMissing         = 1004
	CodeUnknownClient        = 1005
	CodeMaxChunkExceeded     = 1006 // 5000-element limit
	CodeEmptyRequest         = 1007
	CodeIncorrectRequest     = 1008
	CodeMaxChunkExceeded500  = 1009 // 500-element limit (metadata endpoints)
	CodeTokenInvalid         = 1010
	CodeIncorrectContentType = 1011

	// Parameter / lineup / station (2xxx)
	CodeRequiredRequestMissing  = 2002
	CodeMissingCountry          = 2004
	CodeMissingPostalCode       = 2005
	CodeMissingPersonOrNameID   = 2020
	CodeInvalidParameterCountry = 2050
	CodeInvalidUnknownRequest   = 2054
	CodeInvalidDebug            = 2055
	CodeDuplicateLineup         = 2100
	CodeLineupNotFound          = 2101
	CodeUnknownLineup           = 2102
	CodeInvalidLineupDelete     = 2103
	CodeLineupWrongFormat       = 2104
	CodeLineupDeleted           = 2106
	CodeInvalidCountry          = 2108
	CodeInvalidPersonID         = 2109
	CodeStationIDNotFound       = 2200
	CodeStationIDDeleted        = 2201

	// Service status (3xxx)
	CodeServiceOffline = 3000
	CodeServerBusy     = 3001

	// Account auth/state (4xxx)
	CodeAccountExpired          = 4001
	CodeInvalidHash             = 4002
	CodeInvalidUser             = 4003
	CodeAccountLockout          = 4004
	CodeJSONAccessDisabled      = 4005
	CodeTokenExpired            = 4006
	CodeApplicationDisabled     = 4007
	CodeAccountInactive         = 4008
	CodeTooManyLogins           = 4009
	CodeTooManyUniqueIPs        = 4010
	CodeMaxLineupChangesReached = 4100
	CodeMaxLineups              = 4101
	CodeNoLineups               = 4102

	// Image quotas (5xxx)
	CodeImageNotFound          = 5000
	CodeMaxImageDownloads      = 5002
	CodeMaxImageDownloadsTrial = 5003
	CodeMaxImageInvalidURIs    = 5004

	// Programs (6xxx)
	CodeInvalidProgramID  = 6000
	CodeProgramIDQueued   = 6001 // soft retry; was previously surfaced as PROGRAM_GENERATING
	CodeFutureProgram     = 6002

	// Schedules (7xxx)
	CodeScheduleNotFound       = 7000
	CodeInvalidScheduleRequest = 7010
	CodeScheduleRangeExceeded  = 7020
	CodeScheduleNotInLineup    = 7030
	CodeScheduleQueued         = 7100

	// Catastrophic
	CodeHCF = 9999
)
