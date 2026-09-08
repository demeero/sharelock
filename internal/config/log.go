package config

const defaultLogLevel = "info"

type LogConfig struct {
	Level     string
	AddSource bool
}

func loadLog() (LogConfig, error) {
	level := Env("LOG_LEVEL", defaultLogLevel)
	addSource, err := EnvBool("LOG_ADD_SOURCE", false)
	if err != nil {
		return LogConfig{}, err
	}

	return LogConfig{
		Level:     level,
		AddSource: addSource,
	}, nil
}
