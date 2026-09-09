package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadHTTP_UsesDefaults(t *testing.T) {
	unsetConfigEnv(t)

	cfg, err := loadHTTP()

	require.NoError(t, err)
	assert.Equal(t, HTTPConfig{
		Addr:              defaultHTTPAddr,
		ReadHeaderTimeout: defaultHTTPReadHeaderTimeout,
		ReadTimeout:       defaultHTTPReadTimeout,
		WriteTimeout:      defaultHTTPWriteTimeout,
		IdleTimeout:       defaultHTTPIdleTimeout,
		ShutdownTimeout:   defaultHTTPShutdownTimeout,
	}, cfg)
}

func TestLoadHTTP_UsesEnvironmentValues(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("HTTP_ADDR", "127.0.0.1:8443")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "2s")
	t.Setenv("HTTP_READ_TIMEOUT", "3s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "4s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "5s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "6s")
	t.Setenv("TLS_CERT_FILE", "cert.pem")
	t.Setenv("TLS_CERT_KEY_FILE", "key.pem")
	t.Setenv("HTTP_DISABLE_API_DOCS", "true")

	cfg, err := loadHTTP()

	require.NoError(t, err)
	assert.Equal(t, HTTPConfig{
		Addr:              "127.0.0.1:8443",
		TLSCertFile:       "cert.pem",
		TLSKeyFile:        "key.pem",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      4 * time.Second,
		IdleTimeout:       5 * time.Second,
		ShutdownTimeout:   6 * time.Second,
		DisableAPIDocs:    true,
	}, cfg)
}

func TestLoadHTTP_ReturnsParsingError(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("HTTP_READ_TIMEOUT", "soon")

	_, err := loadHTTP()

	require.ErrorContains(t, err, "HTTP_READ_TIMEOUT must be a duration")
}

func TestLoadHTTP_ReturnsErrorForInvalidDisableAPIDocs(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("HTTP_DISABLE_API_DOCS", "not-a-bool")

	_, err := loadHTTP()

	require.ErrorContains(t, err, `env "HTTP_DISABLE_API_DOCS" must be a boolean`)
}

func TestHTTPConfigValidate(t *testing.T) {
	validConfig := HTTPConfig{
		Addr:              ":8080",
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
		ShutdownTimeout:   time.Second,
	}

	tests := []struct {
		name        string
		errorString string
		cfg         HTTPConfig
	}{
		{name: "valid", cfg: validConfig},
		{
			name: "missing required values",
			cfg:  HTTPConfig{},
			errorString: "HTTP configuration errors:\n- HTTP_ADDR is required\n- " +
				"HTTP_READ_HEADER_TIMEOUT must be a positive duration\n- HTTP_READ_TIMEOUT must be a positive duration\n- " +
				"HTTP_SHUTDOWN_TIMEOUT must be a positive duration\n- HTTP_IDLE_TIMEOUT must be a positive duration\n- " +
				"HTTP_WRITE_TIMEOUT must be a positive duration",
		},
		{
			name: "only TLS certificate is configured",
			cfg: HTTPConfig{
				Addr:              validConfig.Addr,
				ReadHeaderTimeout: time.Second,
				ReadTimeout:       time.Second,
				WriteTimeout:      time.Second,
				IdleTimeout:       time.Second,
				ShutdownTimeout:   time.Second,
				TLSCertFile:       "cert.pem",
			},
			errorString: "both TLS_CERT_FILE and TLS_CERT_KEY_FILE must be set together",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.validate()

			if test.errorString == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, test.errorString)
		})
	}
}
