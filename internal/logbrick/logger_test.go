package logbrick

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLevel_ParsesValidLevel(t *testing.T) {
	lvl := ParseLevel("debug", slog.LevelInfo)

	assert.Equal(t, slog.LevelDebug, lvl)
}

func TestParseLevel_UsesFallbackForInvalidLevel(t *testing.T) {
	lvl := ParseLevel("not-a-level", slog.LevelWarn)

	assert.Equal(t, slog.LevelWarn, lvl)
}

func TestParseLevel_UsesFallbackForEmptyLevel(t *testing.T) {
	lvl := ParseLevel("", slog.LevelError)

	assert.Equal(t, slog.LevelError, lvl)
}

func TestConfigure_SetsLevelThreshold(t *testing.T) {
	restoreDefaultLogger(t)

	Configure("warn", false)

	assert.False(t, slog.Default().Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, slog.Default().Enabled(t.Context(), slog.LevelWarn))
}

func TestConfigure_FallsBackToInfoForInvalidLevel(t *testing.T) {
	restoreDefaultLogger(t)

	Configure("not-a-level", false)

	assert.True(t, slog.Default().Enabled(t.Context(), slog.LevelInfo))
	assert.False(t, slog.Default().Enabled(t.Context(), slog.LevelDebug))
}

func TestConfigure_AddsContextAttributesToOutput(t *testing.T) {
	restoreDefaultLogger(t)
	readStdout := captureStdout(t)

	Configure("info", false)
	ctx := AppendCtx(t.Context(), slog.String("request_id", "abc-123"))
	slog.InfoContext(ctx, "handled request")

	out := readStdout()
	assert.Contains(t, out, "request_id=abc-123")
	assert.Contains(t, out, `msg="handled request"`)
}

func restoreDefaultLogger(t *testing.T) {
	t.Helper()

	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func captureStdout(t *testing.T) func() string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	orig := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	return func() string {
		require.NoError(t, w.Close())
		os.Stdout = orig

		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		require.NoError(t, err)

		return buf.String()
	}
}
