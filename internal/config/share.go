package config

import "time"

const (
	defaultMaxShareBytes  = 10 << 20
	defaultMaxTTL         = 30 * 24 * time.Hour
	defaultIdentifierSize = 32
)

type ShareConfig struct {
	MaxEncryptedBytes uint
	MaxTTL            time.Duration
	IdentifierSize    uint
	VacuumInterval    time.Duration
}

func loadShare() (ShareConfig, error) {
	maxEncryptedBytes, err := EnvUint("SHARE_MAX_ENCRYPTED_BYTES", defaultMaxShareBytes)
	if err != nil {
		return ShareConfig{}, err
	}
	maxTTL, err := EnvDuration("SHARE_MAX_TTL", defaultMaxTTL)
	if err != nil {
		return ShareConfig{}, err
	}
	vacuumInterval, err := EnvDuration("SHARE_VACUUM_INTERVAL", time.Hour)
	if err != nil {
		return ShareConfig{}, err
	}
	identifierSize, err := EnvUint("SHARE_IDENTIFIER_SIZE", defaultIdentifierSize)
	if err != nil {
		return ShareConfig{}, err
	}

	cfg := ShareConfig{
		MaxEncryptedBytes: maxEncryptedBytes,
		MaxTTL:            maxTTL,
		IdentifierSize:    identifierSize,
		VacuumInterval:    vacuumInterval,
	}

	return cfg, cfg.validate()
}

func (cfg ShareConfig) validate() error {
	var validationErrors []string

	if cfg.MaxTTL <= 0 {
		validationErrors = append(validationErrors, "SHARE_MAX_TTL must be a positive duration")
	}
	if cfg.VacuumInterval <= 0 {
		validationErrors = append(validationErrors, "SHARE_VACUUM_INTERVAL must be a positive duration")
	}
	if cfg.IdentifierSize <= 0 {
		validationErrors = append(validationErrors, "SHARE_IDENTIFIER_SIZE must be a positive integer")
	}
	if cfg.MaxEncryptedBytes <= 0 {
		validationErrors = append(validationErrors, "SHARE_MAX_ENCRYPTED_BYTES must be a positive integer")
	}

	return validate("Share configuration", validationErrors)
}
