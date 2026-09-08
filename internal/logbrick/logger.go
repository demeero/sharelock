package logbrick

import (
	"log/slog"
	"os"
)

func Configure(level string, addSource bool) {
	lvl := ParseLevel(level, slog.LevelInfo)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl, AddSource: addSource})
	logger := slog.New(NewContextHandler(h))

	slog.SetDefault(logger)
}

func ParseLevel(level string, fallback slog.Level) slog.Level {
	logLvl := &slog.LevelVar{}
	if err := logLvl.UnmarshalText([]byte(level)); err != nil {
		slog.Error("failed parse log level - use fallback",
			"err", err, "level", level, "fallback", fallback.String())
		logLvl.Set(fallback)
	}

	return logLvl.Level()
}
