package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadShare_UsesDefaults(t *testing.T) {
	unsetConfigEnv(t)

	cfg, err := loadShare()

	require.NoError(t, err)
	assert.Equal(t, ShareConfig{
		MaxEncryptedBytes: defaultMaxShareBytes,
		MaxTTL:            defaultMaxTTL,
		IdentifierSize:    defaultIdentifierSize,
		VacuumInterval:    time.Hour,
	}, cfg)
}

func TestLoadShare_UsesEnvironmentValues(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("SHARE_MAX_ENCRYPTED_BYTES", "2048")
	t.Setenv("SHARE_MAX_TTL", "48h")
	t.Setenv("SHARE_VACUUM_INTERVAL", "30m")
	t.Setenv("SHARE_IDENTIFIER_SIZE", "16")

	cfg, err := loadShare()

	require.NoError(t, err)
	assert.Equal(t, ShareConfig{
		MaxEncryptedBytes: 2048,
		MaxTTL:            48 * time.Hour,
		IdentifierSize:    16,
		VacuumInterval:    30 * time.Minute,
	}, cfg)
}

func TestLoadShare_ReturnsParsingError(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("SHARE_IDENTIFIER_SIZE", "large")

	_, err := loadShare()

	require.ErrorContains(t, err, `env "SHARE_IDENTIFIER_SIZE" must be an unsigned integer`)
}

func TestShareConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		errorString string
		cfg         ShareConfig
	}{
		{
			name: "valid",
			cfg: ShareConfig{
				MaxEncryptedBytes: 1,
				MaxTTL:            time.Second,
				IdentifierSize:    1,
				VacuumInterval:    time.Second,
			},
		},
		{
			name: "invalid values",
			cfg:  ShareConfig{},
			errorString: "Share configuration errors:\n- SHARE_MAX_TTL must be a positive duration\n- " +
				"SHARE_VACUUM_INTERVAL must be a positive duration\n- SHARE_IDENTIFIER_SIZE must be a positive integer\n- " +
				"SHARE_MAX_ENCRYPTED_BYTES must be a positive integer",
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
