package chilogger

import (
	"encoding/json"
	"strings"
)

// sensitiveBodyKeys are JSON object keys whose values are redacted when logging
// request/response bodies. It reuses the query redaction list plus signature keys.
var sensitiveBodyKeys = map[string]struct{}{
	"token": {}, "access_token": {}, "api_key": {}, "key": {},
	"password": {}, "code": {}, "state": {}, "authorization": {},
	"secret": {}, "client_secret": {}, "signature": {}, "sign": {},
}

// isBodyLoggable reports whether a body of the given content type should be
// logged. Only textual/JSON payloads qualify; multipart and binary are skipped.
func isBodyLoggable(contentType string) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "multipart/") {
		return false
	}
	return isJSONContentType(ct) ||
		strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "application/x-www-form-urlencoded")
}

func isJSONContentType(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/json") || strings.Contains(ct, "+json")
}

// captureBody prepares a raw body for logging: it redacts sensitive keys in
// valid JSON, then truncates to max bytes with a marker. Non-JSON or invalid
// JSON (e.g. a body already truncated during capture) is logged as-is up to max.
func captureBody(raw []byte, max int, contentType string) string {
	if len(raw) == 0 {
		return ""
	}
	out := string(raw)
	if isJSONContentType(contentType) {
		if redacted, ok := redactJSONBody(raw); ok {
			out = redacted
		}
	}
	if len(out) > max {
		out = out[:max] + "...[truncated]"
	}
	return out
}

// redactJSONBody parses raw as JSON, replaces sensitive values with [redacted],
// and re-marshals. The bool is false when raw is not valid JSON.
func redactJSONBody(raw []byte) (string, bool) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", false
	}
	redactValue(v)
	out, err := json.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(out), true
}

func redactValue(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k, val := range t {
			if _, sensitive := sensitiveBodyKeys[strings.ToLower(k)]; sensitive {
				t[k] = "[redacted]"
			} else {
				redactValue(val)
			}
		}
	case []interface{}:
		for _, val := range t {
			redactValue(val)
		}
	}
}
