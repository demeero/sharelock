package logbrick

import (
	"context"
	"log/slog"
)

type ctxKey string

const (
	slogFields ctxKey = "slog_fields"
)

type ctxHandler struct {
	slog.Handler
}

// NewContextHandler decorates a handler so records inherit attributes stored in their context.
func NewContextHandler(handler slog.Handler) slog.Handler {
	return ctxHandler{Handler: handler}
}

// Handle adds contextual attributes to the Record before calling the underlying
// handler.
func (h ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	if attrs, ok := ctx.Value(slogFields).([]slog.Attr); ok {
		for _, v := range attrs {
			r.AddAttrs(v)
		}
	}

	return h.Handler.Handle(ctx, r)
}

// AppendCtx adds an slog attribute to the provided context so that it will be
// included in any Record created with such context.
func AppendCtx(parent context.Context, attrs ...slog.Attr) context.Context {
	if parent == nil {
		parent = context.Background()
	}

	if v, ok := parent.Value(slogFields).([]slog.Attr); ok {
		v = append(v, attrs...)
		return context.WithValue(parent, slogFields, v)
	}

	v := make([]slog.Attr, 0, len(attrs))
	v = append(v, attrs...)

	return context.WithValue(parent, slogFields, v)
}
