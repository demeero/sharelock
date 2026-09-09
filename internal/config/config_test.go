package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_UsesExplicitEnvFile(t *testing.T) {
	unsetConfigEnv(t)
	envFilePath := filepath.Join(t.TempDir(), "sharelock.env")
	require.NoError(t, os.WriteFile(envFilePath, []byte("LOG_LEVEL=debug\n"), 0o600))

	t.Setenv("ENV_FILE", envFilePath)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoad_ReturnsErrorForMissingExplicitEnvFile(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))

	_, err := Load()
	require.ErrorContains(t, err, "loading env file: failed to open dot env file")
}

func TestLoad_UsesDotEnvFromCurrentDirectory(t *testing.T) {
	unsetConfigEnv(t)
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, ".env"), []byte("LOG_LEVEL=debug\n"), 0o600))
	t.Chdir(directory)

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "debug", cfg.Log.Level)
}

func TestLoad_UsesEnvironmentWhenDotEnvIsMissing(t *testing.T) {
	unsetConfigEnv(t)
	t.Chdir(t.TempDir())

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, defaultLogLevel, cfg.Log.Level)
}

func TestValidate_ReturnsCombinedValidationErrors(t *testing.T) {
	err := validate("component", []string{"first problem", "second problem"})

	require.Error(t, err)
	require.EqualError(t, err, "component errors:\n- first problem\n- second problem")
	assert.NoError(t, validate("component", nil))
}

func unsetConfigEnv(t *testing.T) {
	t.Helper()

	for _, key := range []string{
		"ENV_FILE",
		"LOG_LEVEL",
		"LOG_ADD_SOURCE",
		"DB_PATH",
		"DB_STARTUP_TIMEOUT",
		"DB_MAX_OPEN_CONNS",
		"DB_MAX_IDLE_CONNS",
		"HTTP_ADDR",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"HTTP_DISABLE_API_DOCS",
		"TLS_CERT_FILE",
		"TLS_CERT_KEY_FILE",
		"SHARE_MAX_ENCRYPTED_BYTES",
		"SHARE_MAX_TTL",
		"SHARE_MAX_VIEWS",
		"SHARE_VACUUM_INTERVAL",
		"SHARE_IDENTIFIER_SIZE",
	} {
		t.Setenv(key, "")
	}
}
