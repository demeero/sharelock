package config

import "time"

const (
	defaultDBPath           = "./data/sharelock.db"
	defaultDBStartupTimeout = 10 * time.Second
)

type DBConfig struct {
	Path           string
	MaxOpenConns   uint
	MaxIdleConns   uint
	StartupTimeout time.Duration
}

func loadDB() (DBConfig, error) {
	startupTimeout, err := EnvDuration("DB_STARTUP_TIMEOUT", defaultDBStartupTimeout)
	if err != nil {
		return DBConfig{}, err
	}
	maxOpenConns, err := EnvUint("DB_MAX_OPEN_CONNS", 4)
	if err != nil {
		return DBConfig{}, err
	}
	maxIdleConns, err := EnvUint("DB_MAX_IDLE_CONNS", 4)
	if err != nil {
		return DBConfig{}, err
	}

	cfg := DBConfig{
		Path:           Env("DB_PATH", defaultDBPath),
		MaxOpenConns:   maxOpenConns,
		MaxIdleConns:   maxIdleConns,
		StartupTimeout: startupTimeout,
	}

	return cfg, cfg.validate()
}

func (cfg DBConfig) validate() error {
	var validationErrors []string

	if cfg.Path == "" {
		validationErrors = append(validationErrors, "DB_PATH is required")
	}
	if cfg.StartupTimeout <= 0 {
		validationErrors = append(validationErrors, "DB_STARTUP_TIMEOUT must be a positive duration")
	}
	if cfg.MaxOpenConns == 0 {
		validationErrors = append(validationErrors, "DB_MAX_OPEN_CONNS must be a positive integer")
	}
	if cfg.MaxIdleConns == 0 {
		validationErrors = append(validationErrors, "DB_MAX_IDLE_CONNS must be a positive integer")
	}

	return validate("database configuration", validationErrors)
}
