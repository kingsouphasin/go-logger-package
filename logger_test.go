package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGlobalFunctionsDoNotPanic(t *testing.T) {
	assert.NotPanics(t, func() { Debug("debug msg") })
	assert.NotPanics(t, func() { Info("info msg") })
	assert.NotPanics(t, func() { Warn("warn msg") })
	assert.NotPanics(t, func() { Error("error msg") })
	assert.NotPanics(t, func() { Debugw("debug sugared", "key", "val") })
	assert.NotPanics(t, func() { Infow("info sugared", "key", "val") })
	assert.NotPanics(t, func() { Warnw("warn sugared", "key", "val") })
	assert.NotPanics(t, func() { Errorw("error sugared", "key", "val") })
}

func TestGlobalWithReturnsLogger(t *testing.T) {
	child := With(zap.String("service", "test"))
	assert.NotNil(t, child)
}

func TestGlobalNamedReturnsLogger(t *testing.T) {
	named := Named("component")
	assert.NotNil(t, named)
}

func TestGlobalSetLevel(t *testing.T) {
	require.NoError(t, SetLevel("debug"))
	require.NoError(t, SetLevel("info"))
	assert.Error(t, SetLevel("badlevel"))
}

func TestSetDefault(t *testing.T) {
	cfg := Config{Env: "production", Level: "warn", Console: true, File: false}
	custom, err := New(cfg)
	require.NoError(t, err)

	SetDefault(custom)
	assert.NotPanics(t, func() { Info("via custom default") })
	_ = Sync()
}
