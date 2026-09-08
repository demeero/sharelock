package main

import (
	"errors"
	"log/slog"
	"net/http"
	"time"
	"uuid"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/demeero/sharelock/internal/errbrick"
	"github.com/demeero/sharelock/internal/logbrick"
)

const requestIDHeader = "X-Request-ID"

type apiContainer struct {
	Shares *huma.Group
}

func setupAPI(ver string, mux humago.Mux) apiContainer {
	configureErrorHandler()

	apiConfig := huma.DefaultConfig("Sharelock API", ver)

	api := humago.New(mux, apiConfig)

	apiGroup := huma.NewGroup(api, "/api/v1")
	apiGroup.UseMiddleware(apiInstrumentation)

	shareGroup := huma.NewGroup(apiGroup, "/shares")
	shareGroup.UseSimpleModifier(func(o *huma.Operation) {
		o.Tags = []string{"shares"}
	})

	return apiContainer{
		Shares: shareGroup,
	}
}

func configureErrorHandler() {
	huma.NewErrorWithContext = func(
		ctx huma.Context,
		status int,
		message string,
		errs ...error,
	) huma.StatusError {
		err := errors.Join(errs...)

		switch {
		case errors.Is(err, errbrick.ErrInvalidData):
			slog.InfoContext(ctx.Context(), "invalid data", "err", err)
			return huma.NewError(http.StatusBadRequest, err.Error())

		case errors.Is(err, errbrick.ErrNotFound):
			slog.InfoContext(ctx.Context(), "not found", "err", err)
			return huma.NewError(http.StatusNotFound, err.Error())

		case status < http.StatusInternalServerError:
			return huma.NewError(status, message, errs...)

		default:
			slog.ErrorContext(ctx.Context(), "unhandled error", "err", err)
			return huma.NewError(
				http.StatusInternalServerError,
				"internal error",
			)
		}
	}
}

// apiInstrumentation adds request-scoped log fields and logs request lifecycle events.
func apiInstrumentation(ctx huma.Context, next func(huma.Context)) {
	reqID := uuid.NewV7().String()
	reqCtx := logbrick.AppendCtx(ctx.Context(), requestLogAttrs(ctx, reqID)...)

	ctx = huma.WithContext(ctx, reqCtx)
	ctx.SetHeader(requestIDHeader, reqID)

	startedAt := time.Now()
	slog.InfoContext(ctx.Context(), "http request started")

	next(ctx)

	status := ctx.Status()
	if status == 0 {
		status = http.StatusOK
	}

	duration := time.Since(startedAt).Round(time.Millisecond)

	slog.InfoContext(ctx.Context(), "http request completed",
		"http.status_code", status,
		"http.duration", duration,
	)

	if status > http.StatusRequestHeaderFieldsTooLarge {
		slog.ErrorContext(ctx.Context(), "http request failed",
			"http.status_code", status,
			"http.duration", duration,
		)
	}
}

func requestLogAttrs(ctx huma.Context, requestID string) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("req_id", requestID),
		slog.String("http.method", ctx.Method()),
		slog.String("client.addr", ctx.RemoteAddr()),
	}

	op := ctx.Operation()
	if op == nil {
		return attrs
	}

	attrs = append(attrs, slog.String("http.route", op.Path))
	if op.OperationID != "" {
		attrs = append(attrs, slog.String("op", op.OperationID))
	}

	return attrs
}
