# HTTP Body Logging — Design

**Date:** 2026-07-10
**Status:** Approved

## Goal

Add opt-in logging of HTTP request and response **body content** to all five
middleware modules (net/http, chi, gin, echo, fiber). Today the middleware logs
only metadata (method, path, status, sizes, latency, request_id); the actual
body content is never captured.

## Decisions (locked)

- **Activation:** opt-in via config. Default off. No cost or risk unless enabled.
- **Size cap:** log up to `MaxBodyBytes` (default 4096), then append `...[truncated]`.
- **Content types:** `application/json`, `text/*`, `application/x-www-form-urlencoded`
  only. Multipart/binary skipped (uploads are already logged as file metadata).
- **Redaction:** JSON bodies have sensitive keys redacted (reuse existing
  `sensitiveQueryKeys`, extended with `signature` and `sign`). Non-JSON logged
  as-is up to the cap.

## Config (core package)

Two new `Config` fields, env-driven like the rest:

| Field | Type | Default | Env var |
|-------|------|---------|---------|
| `LogBody` | bool | `false` | `LOGGER_LOG_BODY` |
| `MaxBodyBytes` | int | `4096` | `LOGGER_MAX_BODY_BYTES` |

`LoadConfig()` reads both env vars. `defaultConfig()` sets the defaults.

Core exposes an accessor so the separate middleware modules can read the setting
without parsing env themselves:

```go
// BodyConfig reports whether request/response bodies should be logged and the
// maximum number of bytes to log per body.
func BodyConfig() (enabled bool, maxBytes int)
```

Backed by package-level values set when a logger is built from a `Config`
(same mechanism as the existing global default). Since `init()` and any explicit
`New(LoadConfig())` read the same env, the values are stable.

## New log fields

- `request_body` — added to the `HTTP Request` log entry.
- `response_body` — added to the `HTTP Response` log entry.

Each field is included only when: `LogBody=true`, the body is non-empty, and the
content-type qualifies. Otherwise the field is omitted entirely (no empty noise).

## Shared capture logic (one helper per module)

The middleware modules are independent Go modules and already duplicate small
helpers (`sanitizeQuery`, `uploadedFile`, `clientIP`). Body helpers follow the
same pattern — duplicated per module, identical logic:

```go
// captureBody redacts (JSON only) then truncates a raw body for logging.
func captureBody(raw []byte, max int, contentType string) string {
    if len(raw) == 0 {
        return ""
    }
    var out string
    if isJSONContentType(contentType) {
        out = redactJSONBody(raw) // parse full, redact sensitive keys, re-marshal
    } else {                       // falls back to string(raw) if unparseable
        out = string(raw)
    }
    if len(out) > max {
        out = out[:max] + "...[truncated]"
    }
    return out
}
```

- `isBodyLoggable(contentType)` — true for json / text/* / x-www-form-urlencoded,
  false for multipart and everything else.
- `redactJSONBody(raw)` — unmarshal into `interface{}`, walk maps recursively,
  replace values whose key is in `sensitiveQueryKeys` (extended with `signature`,
  `sign`) with `[redacted]`, marshal back. On unmarshal error, return
  `string(raw)` (still truncated downstream). Key match is case-insensitive.
- **Order matters:** redact the full body first, then truncate. Truncated JSON
  would not parse.

## Per-framework capture (the only differing part)

| Framework | Request body | Response body |
|-----------|-------------|---------------|
| net/http, chi | `io.ReadAll(r.Body)`, then `r.Body = io.NopCloser(bytes.NewReader(buf))` so the handler still reads it | extend existing `responseWriter` with a capped `bytes.Buffer`; capture in `Write` |
| gin | read + restore `c.Request.Body` | wrap `c.Writer` with a tee-to-buffer `gin.ResponseWriter` |
| echo | read + restore `c.Request().Body` | wrap `c.Response().Writer` with a tee-to-buffer `http.ResponseWriter` |
| fiber | `c.Body()` directly — fasthttp already buffers, no restore | `c.Response().Body()` after `Next()` |

Request body is captured **before** the handler runs (and logged on `HTTP Request`);
response body **after** (logged on `HTTP Response`).

## Tradeoffs / non-goals

- Enabling request-body logging reads the full request body into memory (to
  restore it for the handler). Fine for JSON APIs. **No hard memory cap** in this
  version (YAGNI); only the logged portion is capped at `MaxBodyBytes`.
- Response-body capture buffers up to `MaxBodyBytes` only (the tee stops copying
  past the cap), so large responses do not blow up memory.
- Redaction is key-based on JSON only. Non-JSON secrets (e.g. a raw token in a
  text body) are not redacted — documented behavior.

## Testing

Table tests per module:

- Nested JSON sensitive keys are redacted.
- Body over cap is truncated with the marker.
- Multipart request → no `request_body` field (file metadata only, as today).
- Handler still receives the **full** request body after middleware reads it.
- `LogBody=false` (default) → no `request_body`/`response_body` fields at all.
- Non-JSON text body logged verbatim up to cap.

## Backward compatibility

- Default off → zero behavior change for existing users.
- No signature changes to `Middleware()` / `Handle()`.
- New config fields are additive.
