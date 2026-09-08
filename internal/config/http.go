package config

import (
	"errors"
	"time"
)

const (
	defaultHTTPAddr              = ":8080"
	defaultHTTPReadHeaderTimeout = 10 * time.Second
	defaultHTTPReadTimeout       = 30 * time.Second
	defaultHTTPWriteTimeout      = 30 * time.Second
	defaultHTTPIdleTimeout       = 60 * time.Second
	defaultHTTPShutdownTimeout   = 10 * time.Second
)

type HTTPConfig struct {
	Addr              string
	TLSCertFile       string
	TLSKeyFile        string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func loadHTTP() (HTTPConfig, error) {
	readHeaderTimeout, err := EnvDuration("HTTP_READ_HEADER_TIMEOUT", defaultHTTPReadHeaderTimeout)
	if err != nil {
		return HTTPConfig{}, err
	}
	readTimeout, err := EnvDuration("HTTP_READ_TIMEOUT", defaultHTTPReadTimeout)
	if err != nil {
		return HTTPConfig{}, err
	}
	writeTimeout, err := EnvDuration("HTTP_WRITE_TIMEOUT", defaultHTTPWriteTimeout)
	if err != nil {
		return HTTPConfig{}, err
	}
	idleTimeout, err := EnvDuration("HTTP_IDLE_TIMEOUT", defaultHTTPIdleTimeout)
	if err != nil {
		return HTTPConfig{}, err
	}
	shutdownTimeout, err := EnvDuration("HTTP_SHUTDOWN_TIMEOUT", defaultHTTPShutdownTimeout)
	if err != nil {
		return HTTPConfig{}, err
	}

	cfg := HTTPConfig{
		Addr:              Env("HTTP_ADDR", defaultHTTPAddr),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ShutdownTimeout:   shutdownTimeout,
		TLSCertFile:       Env("TLS_CERT_FILE", ""),
		TLSKeyFile:        Env("TLS_CERT_KEY_FILE", ""),
	}

	return cfg, cfg.validate()
}

func (cfg HTTPConfig) validate() error {
	var validationErrors []string

	if cfg.Addr == "" {
		validationErrors = append(validationErrors, "HTTP_ADDR is required")
	}
	if cfg.ReadHeaderTimeout <= 0 {
		validationErrors = append(validationErrors, "HTTP_READ_HEADER_TIMEOUT must be a positive duration")
	}
	if cfg.ReadTimeout <= 0 {
		validationErrors = append(validationErrors, "HTTP_READ_TIMEOUT must be a positive duration")
	}
	if cfg.ShutdownTimeout <= 0 {
		validationErrors = append(validationErrors, "HTTP_SHUTDOWN_TIMEOUT must be a positive duration")
	}
	if cfg.IdleTimeout <= 0 {
		validationErrors = append(validationErrors, "HTTP_IDLE_TIMEOUT must be a positive duration")
	}
	if cfg.WriteTimeout <= 0 {
		validationErrors = append(validationErrors, "HTTP_WRITE_TIMEOUT must be a positive duration")
	}
	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return errors.New("both TLS_CERT_FILE and TLS_CERT_KEY_FILE must be set together")
	}

	return validate("HTTP configuration", validationErrors)
}
