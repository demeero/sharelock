package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnv_UsesTrimmedValueOrFallback(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "value", value: "  configured value  ", fallback: "fallback", want: "configured value"},
		{name: "blank value", value: "  ", fallback: "fallback", want: "fallback"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFIG_TEST_STRING", test.value)

			assert.Equal(t, test.want, Env("CONFIG_TEST_STRING", test.fallback))
		})
	}
}

func TestEnvDuration(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		errorString string
		want        time.Duration
	}{
		{name: "fallback", want: time.Second},
		{name: "parsed", value: "2m", want: 2 * time.Minute},
		{name: "invalid", value: "tomorrow", errorString: "CONFIG_TEST_DURATION must be a duration"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFIG_TEST_DURATION", test.value)

			got, err := EnvDuration("CONFIG_TEST_DURATION", time.Second)

			if test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestEnvUint(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		errorString string
		want        uint
	}{
		{name: "fallback", want: 3},
		{name: "parsed", value: "42", want: 42},
		{name: "negative", value: "-1", errorString: `env "CONFIG_TEST_UINT" must be an unsigned integer`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFIG_TEST_UINT", test.value)

			got, err := EnvUint("CONFIG_TEST_UINT", 3)

			if test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		errorString string
		want        int
	}{
		{name: "fallback", want: 3},
		{name: "parsed", value: "-42", want: -42},
		{name: "invalid", value: "many", errorString: `env "CONFIG_TEST_INT" must be an integer`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFIG_TEST_INT", test.value)

			got, err := EnvInt("CONFIG_TEST_INT", 3)

			if test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		errorString string
		want        bool
	}{
		{name: "fallback", want: true},
		{name: "parsed", value: "false", want: false},
		{name: "invalid", value: "sometimes", errorString: `env "CONFIG_TEST_BOOL" must be a boolean`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFIG_TEST_BOOL", test.value)

			got, err := EnvBool("CONFIG_TEST_BOOL", true)

			if test.errorString != "" {
				require.ErrorContains(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestTryLoadDotEnv_IgnoresMissingFile(t *testing.T) {
	err := TryLoadDotEnv(filepath.Join(t.TempDir(), "missing.env"))

	require.NoError(t, err)
}

func TestLoadDotEnv_LoadsValuesWithoutOverwritingEnvironment(t *testing.T) {
	envFilePath := filepath.Join(t.TempDir(), "sharelock.env")
	require.NoError(t, os.WriteFile(envFilePath, []byte("# comment\nFROM_FILE = value\nEXISTING=from-file\nMALFORMED\n"), 0o600))
	t.Setenv("FROM_FILE", "")
	t.Setenv("EXISTING", "from-environment")

	require.NoError(t, LoadDotEnv(envFilePath))
	assert.Equal(t, "value", os.Getenv("FROM_FILE"))
	assert.Equal(t, "from-environment", os.Getenv("EXISTING"))
}

func TestLoadDotEnv_ReturnsErrorForMissingFile(t *testing.T) {
	err := LoadDotEnv(filepath.Join(t.TempDir(), "missing.env"))

	require.ErrorContains(t, err, "failed to open dot env file")
}
