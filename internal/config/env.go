package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func Env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func EnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration, for example 5s or 1m", key)
	}

	return duration, nil
}

func EnvUint(key string, fallback uint) (uint, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.ParseUint(v, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("env %q must be an unsigned integer, got %q", key, v)
	}

	return uint(n), nil
}

func EnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("env %q must be an integer, got %q", key, v)
	}

	return n, nil
}

func EnvBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("env %q must be a boolean (1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False), got %q", key, v)
	}

	return b, nil
}

func TryLoadDotEnv(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	return LoadDotEnv(path)
}

func LoadDotEnv(path string) error {
	// #nosec G703 -- ENV_FILE is deployment-controlled startup configuration.
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open dot env file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		// Don't overwrite existing env vars — env takes precedence
		if os.Getenv(strings.TrimSpace(key)) == "" {
			os.Setenv(strings.TrimSpace(key), strings.TrimSpace(val))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read dot env file: %w", err)
	}

	return nil
}
