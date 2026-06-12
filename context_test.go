package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithContextAndFromContext(t *testing.T) {
	cfg := Config{Env: "production", Level: "info", Console: true, File: false}
	log, err := New(cfg)
	require.NoError(t, err)

	ctx := WithContext(context.Background(), log)
	retrieved := FromContext(ctx)
	assert.Equal(t, log, retrieved)
	_ = log.Sync()
}

func TestFromContextReturnsDefaultWhenNotSet(t *testing.T) {
	ctx := context.Background()
	log := FromContext(ctx)
	assert.NotNil(t, log)
}
