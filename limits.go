package schedulesdirect

import "time"

// Documented Schedules Direct rate-limit numerics, batch caps, retry windows,
// and TTLs. Exposed as named constants so consumers don't transcribe published
// SD numbers from their docs into application code.
//
// These are facts about the SD service. The library exposes facts; consumers
// implement the policy mechanism (single-flight, hold maps, soft-caps below
// these hard-caps, daily counters). Symmetric with the Code* constants in
// error.go: those are facts about SD's error-code vocabulary; the values
// below are facts about SD's rate-limit / timing vocabulary.
//
// Source: SD API-20141201 documentation.

// Daily quotas. Counters reset at DailyWindowReset (00:00Z).
const (
	// IPBlockDailyLimit caps GET /ip_isblocked at 100 invocations per 24-hour
	// window. The window resets at 00:00Z.
	IPBlockDailyLimit = 100

	// LineupChangesDailyLimit caps the combined PUT/DELETE /lineups operations
	// at 6 per 24-hour window. The ChangesRemaining field on
	// LineupChangeResponse reports the remaining budget.
	LineupChangesDailyLimit = 6

	// DailyWindowReset is the UTC time-of-day when 24-hour rate-limit windows
	// reset. Applies to /ip_isblocked, lineup-change budget, and per-account
	// image-download quotas.
	DailyWindowReset = "00:00Z"
)

// Per-request batch caps. Exceeding returns CodeMaxChunkExceeded (1006) for
// the 5000-element endpoints or CodeMaxChunkExceeded500 (1009) for the
// 500-element endpoints.
const (
	// BatchMaxEntries is the per-request element cap for endpoints that accept
	// large arrays of programIDs or stationIDs:
	// POST /programs, POST /schedules, POST /schedules/md5.
	BatchMaxEntries = 5000

	// MetadataBatchMaxEntries is the per-request element cap for the metadata
	// endpoints: POST /metadata/description/, POST /metadata/programs/,
	// POST /xref.
	MetadataBatchMaxEntries = 500
)

// Documented retry windows and TTLs.
const (
	// ServiceOfflineRetry is the documented hold duration after receiving
	// SERVICE_OFFLINE (CodeServiceOffline / 3000): clients should not
	// reconnect for 30 minutes.
	ServiceOfflineRetry = 30 * time.Minute

	// AccountLockoutRetry is the documented hold duration after
	// ACCOUNT_LOCKOUT (CodeAccountLockout / 4004) following too-many-login
	// failures: 15 minutes.
	AccountLockoutRetry = 15 * time.Minute

	// SportsStatusQueuedRetry is the documented retry interval after receiving
	// PROGRAMID_QUEUED (CodeProgramIDQueued / 6001) on
	// GET /metadata/stillRunning/{programID}: wait 30 seconds before retrying.
	SportsStatusQueuedRetry = 30 * time.Second

	// TokenValidity is the documented lifetime of a successfully-issued token:
	// 24 hours from the time of first issue. Token requests are themselves
	// rate-limited; consumers should reuse the issued token until its
	// TokenExpires timestamp rather than requesting a fresh one.
	TokenValidity = 24 * time.Hour

	// ImageRedirectTTL is the validity window of the temporary S3 URL returned
	// by the GET /image/{uri} 303 redirect: 120 seconds. Consumers must fetch
	// within this window; the URL is not cacheable.
	ImageRedirectTTL = 120 * time.Second

	// BatchServerTimeout is the server-side timeout on batch endpoints
	// (POST /programs, /schedules, /schedules/md5). The server terminates
	// requests that haven't completed within 10 minutes; consumers on slow
	// connections should chunk below BatchMaxEntries to stay under it.
	BatchServerTimeout = 10 * time.Minute
)
