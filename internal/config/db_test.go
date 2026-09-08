package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDB_UsesDefaults(t *testing.T) {
	unsetConfigEnv(t)

	cfg, err := loadDB()

	require.NoError(t, err)
	assert.Equal(t, DBConfig{
		Path:           defaultDBPath,
		MaxOpenConns:   4,
		MaxIdleConns:   4,
		StartupTimeout: defaultDBStartupTimeout,
	}, cfg)
}

func TestLoadDB_UsesEnvironmentValues(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("DB_PATH", "./testdata/sharelock.db")
	t.Setenv("DB_MAX_OPEN_CONNS", "8")
	t.Setenv("DB_MAX_IDLE_CONNS", "3")
	t.Setenv("DB_STARTUP_TIMEOUT", "20s")

	cfg, err := loadDB()

	require.NoError(t, err)
	assert.Equal(t, DBConfig{
		Path:           "./testdata/sharelock.db",
		MaxOpenConns:   8,
		MaxIdleConns:   3,
		StartupTimeout: 20 * time.Second,
	}, cfg)
}

func TestLoadDB_ReturnsParsingError(t *testing.T) {
	unsetConfigEnv(t)
	t.Setenv("DB_MAX_OPEN_CONNS", "many")

	_, err := loadDB()

	require.ErrorContains(t, err, `env "DB_MAX_OPEN_CONNS" must be an unsigned integer`)
}

func TestDBConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		errorString string
		cfg         DBConfig
		wantError   bool
	}{
		{
			name: "valid",
			cfg: DBConfig{
				Path:           "sharelock.db",
				MaxOpenConns:   1,
				MaxIdleConns:   1,
				StartupTimeout: time.Second,
			},
		},
		{
			name:      "invalid values",
			cfg:       DBConfig{},
			wantError: true,
			errorString: "database configuration errors:\n- DB_PATH is required\n- " +
				"DB_STARTUP_TIMEOUT must be a positive duration\n- DB_MAX_OPEN_CONNS must be a positive integer\n- " +
				"DB_MAX_IDLE_CONNS must be a positive integer",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.validate()

			if test.wantError {
				require.EqualError(t, err, test.errorString)
				return
			}
			require.NoError(t, err)
		})
	}
}
