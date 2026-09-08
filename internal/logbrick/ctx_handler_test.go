package logbrick

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContextHandler_Handle_AddsContextAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(buf, nil)))
	ctx := AppendCtx(t.Context(), slog.String("request_id", "abc-123"))

	logger.InfoContext(ctx, "handled request")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "abc-123", entry["request_id"])
	assert.Equal(t, "handled request", entry["msg"])
}

func TestNewContextHandler_Handle_NoContextAttributes(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(buf, nil)))

	logger.InfoContext(t.Context(), "handled request")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.NotContains(t, entry, "request_id")
	assert.Equal(t, "handled request", entry["msg"])
}

func TestNewContextHandler_Handle_MultipleAttributesPreserveOrder(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(NewContextHandler(slog.NewJSONHandler(buf, nil)))
	ctx := AppendCtx(t.Context(), slog.String("a", "1"), slog.String("b", "2"))

	logger.InfoContext(ctx, "msg")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "1", entry["a"])
	assert.Equal(t, "2", entry["b"])
}

func TestAppendCtx_AccumulatesAttributesAcrossCalls(t *testing.T) {
	ctx := AppendCtx(t.Context(), slog.String("a", "1"))
	ctx = AppendCtx(ctx, slog.String("b", "2"))

	attrs, ok := ctx.Value(slogFields).([]slog.Attr)

	require.True(t, ok)
	require.Len(t, attrs, 2)
	assert.Equal(t, "a", attrs[0].Key)
	assert.Equal(t, "b", attrs[1].Key)
}

func TestAppendCtx_HandlesNilParent(t *testing.T) {
	ctx := AppendCtx(nil, slog.String("a", "1")) //nolint:staticcheck // verifying nil-parent handling

	require.NotNil(t, ctx)
	attrs, ok := ctx.Value(slogFields).([]slog.Attr)
	require.True(t, ok)
	require.Len(t, attrs, 1)
	assert.Equal(t, "a", attrs[0].Key)
}
