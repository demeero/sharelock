// Package config reads Sharelock's process configuration.
package config

import (
	"fmt"
	"os"
	"strings"
)

// buildVersion, buildTime, and commit are replaced by release builds through -ldflags.
var (
	buildVersion = "dev"
	buildTime    = "unknown"
	commit       = "unknown"
)

// Config contains all runtime configuration supplied by the environment.
type Config struct {
	Version   string
	BuildTime string
	Commit    string
	Log       LogConfig
	DB        DBConfig
	HTTP      HTTPConfig
	Share     ShareConfig
}

func Load() (Config, error) {
	envFilePath := os.Getenv("ENV_FILE")
	var err error
	if envFilePath != "" {
		err = LoadDotEnv(envFilePath)
	} else {
		err = TryLoadDotEnv(".env")
	}
	if err != nil {
		return Config{}, fmt.Errorf("loading env file: %w", err)
	}

	logConfig, err := loadLog()
	if err != nil {
		return Config{}, err
	}
	httpConfig, err := loadHTTP()
	if err != nil {
		return Config{}, err
	}
	dbConfig, err := loadDB()
	if err != nil {
		return Config{}, err
	}
	shareConfig, err := loadShare()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Version:   buildVersion,
		BuildTime: buildTime,
		Commit:    commit,
		Log:       logConfig,
		HTTP:      httpConfig,
		DB:        dbConfig,
		Share:     shareConfig,
	}, nil
}

func validate(component string, validationErrors []string) error {
	if len(validationErrors) > 0 {
		return fmt.Errorf("%s errors:\n- %s", component, strings.Join(validationErrors, "\n- "))
	}

	return nil
}
