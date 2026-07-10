package echologger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptureBody_RedactsNestedJSON(t *testing.T) {
	raw := []byte(`{"user":"alice","password":"secret123","nested":{"token":"abc","ok":1}}`)
	out := captureBody(raw, 4096, "application/json")

	assert.Contains(t, out, `"password":"[redacted]"`)
	assert.Contains(t, out, `"token":"[redacted]"`)
	assert.Contains(t, out, `"user":"alice"`)
	assert.NotContains(t, out, "secret123")
	assert.NotContains(t, out, `"abc"`)
}

func TestCaptureBody_RedactsSignature(t *testing.T) {
	raw := []byte(`{"sign":"deadbeef","signature":"cafe","amount":100}`)
	out := captureBody(raw, 4096, "application/json")

	assert.NotContains(t, out, "deadbeef")
	assert.NotContains(t, out, "cafe")
	assert.Contains(t, out, `"amount":100`)
}

func TestCaptureBody_TruncatesWithMarker(t *testing.T) {
	raw := make([]byte, 100)
	for i := range raw {
		raw[i] = 'x'
	}
	out := captureBody(raw, 10, "text/plain")

	assert.Equal(t, "xxxxxxxxxx...[truncated]", out)
}

func TestCaptureBody_NonJSONVerbatim(t *testing.T) {
	raw := []byte("plain text body")
	out := captureBody(raw, 4096, "text/plain")
	assert.Equal(t, "plain text body", out)
}

func TestCaptureBody_InvalidJSONFallsBackToRaw(t *testing.T) {
	raw := []byte(`{"truncated":`)
	out := captureBody(raw, 4096, "application/json")
	assert.Equal(t, `{"truncated":`, out)
}

func TestCaptureBody_Empty(t *testing.T) {
	assert.Equal(t, "", captureBody(nil, 4096, "application/json"))
	assert.Equal(t, "", captureBody([]byte{}, 4096, "application/json"))
}

func TestIsBodyLoggable(t *testing.T) {
	cases := map[string]bool{
		"application/json":                  true,
		"application/json; charset=utf-8":   true,
		"application/vnd.api+json":          true,
		"text/plain":                        true,
		"text/html":                         true,
		"application/x-www-form-urlencoded": true,
		"multipart/form-data; boundary=xy":  false,
		"application/octet-stream":          false,
		"image/png":                         false,
	}
	for ct, want := range cases {
		assert.Equalf(t, want, isBodyLoggable(ct), "content type %q", ct)
	}
}
