# go.schedules-direct

A Go client for the [Schedules Direct](https://www.schedulesdirect.org/) REST API, version `20141201`.

Pure-Go, no CGO, standard library only. Public domain ([Unlicense](LICENSE)).

## Status

Every endpoint in the bundled OpenAPI spec is covered, with response types modeled against captured wire samples (not just the published spec). The spec excludes SD endpoints that are deprecated or empirically non-functional; those omissions are deliberate and not gaps in coverage. The library is wire-protocol-only — token lifecycle, persistence, rate-limiting, observability, and pipeline orchestration are deliberately *not* in scope and are the consumer's responsibility.

## Install

```sh
go get github.com/podgarr/go.schedules-direct
```

Requires Go 1.26 or later.

## Quickstart

```go
package main

import (
    "context"
    "fmt"
    "log"

    sd "github.com/podgarr/go.schedules-direct"
)

func main() {
    ctx := context.Background()

    // User-Agent is mandatory — SD rejects requests without one (error 1003).
    // The library does not synthesize a default; you choose what to send.
    client, err := sd.NewClient("MyApp/1.0")
    if err != nil {
        log.Fatal(err)
    }

    // Authenticate — password must be lowercase sha1 hex.
    tok, err := client.GetToken(ctx, sd.Account{
        Username: "user@example.com",
        Password: sd.HashPassword("plaintext-password"),
    })
    if err != nil {
        log.Fatal(err)
    }
    client.Token = tok.Token

    // Use any endpoint that requires a token.
    status, err := client.GetStatus(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("account expires: %s, max lineups: %d\n",
        status.Account.Expires, status.Account.MaxLineups)
    for _, l := range status.Lineups {
        fmt.Printf("  %s — %s (%s)\n", l.Lineup, l.Name, l.Transport)
    }
}
```

## Scope

The library is the SD wire protocol bound to Go shapes. The hard line:

| In scope | Out of scope (consumer's job) |
|---|---|
| Request and response types | Token storage and refresh |
| HTTP client with token + UA injection | Observability |
| One Go method per SD endpoint | Rate-limiting policy |
| Error code constants + envelope parsing | Single-flight gating |
| Pluggable HTTP transport via `*http.Client` | Database persistence |
| Real-wire test fixtures | Pipeline phase orchestration |
|  | Retry policy beyond what SD documents |

If a feature feels like it should belong but is in the right column, your application is where it goes — not this library. Pick whatever shape fits.

## Endpoint coverage

| Path | Method | Auth |
|---|---|---|
| POST `/token` | `GetToken` | none |
| GET `/version/{clientName}` | `GetVersion` | none |
| GET `/status` | `GetStatus` | token |
| GET `/ip_isblocked` | `GetIPBlockStatus` | none |
| GET `/available` | `GetAvailableServices` | none |
| GET `/available/countries` | `GetAvailableCountries` | none |
| GET `/available/languages` | `GetAvailableLanguages` | none |
| GET `/available/dvb-s` | `GetAvailableDVBS` | none |
| GET `/transmitters/{iso3166alpha3}` | `GetTransmitters` | none |
| GET `/lineups` | `GetLineups` | token |
| GET `/lineups/{lineupID}` | `GetLineupMapping` | token |
| PUT `/lineups/{lineupID}` | `AddLineup` | token |
| DELETE `/lineups/{lineupID}` | `DeleteLineup` | token |
| GET `/lineups/preview/{lineupID}` | `PreviewLineup` | token |
| GET `/headends` | `GetHeadends` | token |
| POST `/schedules/md5` | `GetSchedulesMd5` | token |
| POST `/schedules` | `GetSchedules` | token |
| POST `/programs` | `GetPrograms` | token |
| GET `/image/{uri}` | `GetImageURL` | token |
| POST `/metadata/programs/` | `GetProgramImagesBatch` | token |
| GET `/metadata/programs/{programID}` | `GetProgramImages` | token |
| POST `/metadata/description/` | `GetProgramDescriptions` | token |
| GET `/metadata/stillRunning/{programID}` | `GetSportsEventStatus` | token |
| GET `/celebrity/{personID}` | `GetCelebrity` | token |
| GET `/metadata/celebrity/{personID}` | `GetCelebrityImages` | token |
| POST `/xref` | `GetLanguageCrossReferences` | token |
| DELETE `/messages/{messageID}` | `DeleteMessage` | token |

## Authentication

Tokens are 32-character bearer values issued by `POST /token`. SD documents tokens as valid for 24 hours from issue, and explicitly asks consumers *not* to force a new token before the existing one expires (`Account.NewToken: true`) — token requests are themselves rate-limited.

The library does not manage token lifecycle. The recommended pattern: store the issued `Token.Token` and `Token.TokenExpires` (UNIX epoch) in your consumer's state, set `client.Token` before token-requiring calls, refresh on `*sd.Error` with `Code == sd.CodeTokenExpired`.

## Error handling

Every endpoint method returns a typed response or an error. SD's envelope error (`code != 0`) comes back as `*sd.Error`:

```go
status, err := client.GetStatus(ctx)
if err != nil {
    var sdErr *sd.Error
    if errors.As(err, &sdErr) {
        switch sdErr.Code {
        case sd.CodeTokenExpired:
            // refresh + retry
        case sd.CodeServiceOffline:
            // back off; SD documents 30 min retry
        case sd.CodeMaxImageDownloads:
            // STOP — continued requests will block your account
        }
    }
    // transport / parse failures — wrapped, not *sd.Error
    return err
}
```

The full set of SD error codes is exposed as constants (`CodeOK`, `CodeUserAgentRequired`, `CodeServiceOffline`, `CodeInvalidUser`, etc. — see [`error.go`](error.go) for the complete list). Consumers should check `Code` against named constants rather than magic numbers.

### Per-element errors in batch responses

`GetSchedules`, `GetPrograms`, and `GetProgramImagesBatch` return arrays where individual entries can carry per-element failure inline (e.g. `STATIONID_DELETED`, `INVALID_PROGRAMID`, `SCHEDULE_QUEUED`). The top-level call does *not* return an `*Error` for these — you iterate the array and check `Code` on each entry.

```go
schedules, err := client.GetSchedules(ctx, requests)
if err != nil {
    return err // top-level (auth, network) failure
}
for _, s := range schedules {
    if s.Code != 0 {
        // per-station failure: s.Response, s.RetryTime, s.MinDate/MaxDate, ...
        continue
    }
    // s.Programs is populated
}
```

## HTTP client customization

The `*http.Client` is fully customizable via `WithHTTPClient`. Pass whatever client your application needs — custom timeout, custom transport, anything else `http.Client` exposes:

```go
client, _ := sd.NewClient("MyApp/1.0",
    sd.WithHTTPClient(&http.Client{
        Timeout: 30 * time.Second,
    }),
)
```

Cross-cutting concerns (rate limiting, single-flight, observability) are the consumer's call. The library doesn't prescribe a mechanism — implement at whichever layer fits your application.

## Image fetching

`GetImageURL` does not fetch image bytes. SD responds to `GET /image/{uri}` with an HTTP 303 redirect to a temporary S3 URL (valid 120 seconds); the library reads the `Location` header and returns the URL so you can fetch on your own schedule, with your own caching, gzip handling, and retry policy.

```go
url, err := client.GetImageURL(ctx, imageURI)
if err != nil { /* ... */ }
// url is the S3 URL — fetch with any HTTP client, cache by content hash.
```

If the image URI in a metadata response is already a full S3 URL (`https://...`), fetch it directly without calling `GetImageURL` at all.

**Image quota awareness**: SD imposes a per-account daily image-download limit. Code `5002 MAX_IMAGE_DOWNLOADS` (or `5003` for trial accounts) means stop immediately — continued requests will result in an account block.

## OpenAPI specification

The bundled spec is [`schedulesdirect-api_openapi-v3.0.3.json`](schedulesdirect-api_openapi-v3.0.3.json) — an OpenAPI 3.0.3 description of the Schedules Direct REST API (version `20141201`), synthesized from SD's published markdown and validated against captured wire samples. The library's request and response types track this spec.

If you hit a wire shape this library doesn't model, please open an issue with the captured response so the spec and types can be brought up to date.

## Testing

```sh
go test ./...
```

Tests are httptest-driven against fixtures under [`testdata/`](testdata/). Most fixtures are extracted directly from real Schedules Direct captures; a few (transmitters error case, image redirects) are synthesized from documented behavior.

## Acknowledgments

Two prior Go projects informed the shape of this library, neither as code source:

- [`tellytv/go.schedulesdirect`](https://github.com/tellytv/go.schedulesdirect) (2018, no LICENSE) — library-shape reference for organization (per-endpoint files, types separate from client).
- [`mar-mei/guide2go`](https://github.com/mar-mei/guide2go) (2023, MIT, but `module main`) — SD-handling parity reference; *not* a blueprint. Its package-global `Token` and monolithic `GetData()` are explicitly not adopted; per-instance state on `*Client` keeps token lifecycle a consumer concern without process-global coupling.

## License

[Unlicense](LICENSE) — public domain. Use it for anything.
