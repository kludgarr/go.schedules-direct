// Package schedulesdirect provides a Go client for the Schedules Direct REST API
// (version 20141201).
//
// The library encodes Schedules Direct's wire protocol as Go shapes and methods.
// Multi-account state, persistence, observability, rate limiting, single-flight,
// and pipeline orchestration are out of scope — consumers handle those at
// whichever layer of their application fits.
//
// Schedules Direct requires a User-Agent header on every request; missing or
// generic User-Agents are rejected with error code 1003.
package schedulesdirect

const (
	// APIVersion is the Schedules Direct REST API version this library targets.
	APIVersion = "20141201"

	// BaseURL is the Schedules Direct API base URL for APIVersion.
	BaseURL = "https://json.schedulesdirect.org/" + APIVersion + "/"
)
