package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLog_UsesDefaults(t *testing.T) {
	unsetConfigEnv(t)

	cfg, err := loadLog()

	require.NoError(t, err)
	assert.Equal(t, LogConfig{Level: defaultLogLevel}, cfg)
}

func TestLoadLog_UsesEnvironmentValues(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_ADD_SOURCE", "true")

	cfg, err := loadLog()

	require.NoError(t, err)
	assert.Equal(t, LogConfig{Level: "debug", AddSource: true}, cfg)
}

func TestLoadLog_ReturnsErrorForInvalidAddSource(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("LOG_ADD_SOURCE", "sometimes")

	_, err := loadLog()

	require.ErrorContains(t, err, `env "LOG_ADD_SOURCE" must be a boolean`)
}
