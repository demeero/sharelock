package share

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/demeero/sharelock/internal/config"
	"github.com/demeero/sharelock/internal/errbrick"

	"github.com/danielgtaylor/huma/v2"
)

const maxJSONOverhead = 1 << 20

// createShareReqBody is the browser-visible, encrypted share payload.
type createShareReqBody struct {
	Views            *int64 `doc:"Number of allowed opens. Omit to allow opens until expiration." json:"views,omitempty"`
	Envelope         string `doc:"Opaque, browser-encrypted share envelope."                      json:"envelope"`
	ExpiresInSeconds int64  `doc:"Lifetime in seconds."                                           json:"expires_in_seconds"`
}

// createShareReq is the input for creating an encrypted share.
type createShareReq struct {
	Body createShareReqBody
}

// createShareRespBody is returned once a share is persisted.
type createShareRespBody struct {
	ID          string `doc:"Share identifier to use in the public URL."       json:"id"`
	RevokeToken string `doc:"Secret capability required to revoke this share." json:"revoke_token"`
}

// createShareResp is the response to a successful share creation.
type createShareResp struct {
	Body createShareRespBody
}

// openShareReq identifies the share to retrieve.
type openShareReq struct {
	Body *struct{}
	ID   string `doc:"Share identifier from the URL." path:"id"`
}

// openShareRespBody is the opaque payload returned to the browser.
type openShareRespBody struct {
	ViewsLeft *int64 `doc:"Opens left after this one. Null when there is no limit." json:"views_left"`
	Envelope  string `doc:"Opaque, browser-encrypted share envelope."               json:"envelope"`
	Burned    bool   `doc:"Whether this successful read removed the share."         json:"burned"`
}

// openShareResp is the response to a successful share read.
type openShareResp struct {
	Body openShareRespBody
}

// revokeShareReq identifies a share and supplies its revoke capability.
type revokeShareReq struct {
	ID          string `doc:"Share identifier from the URL."                path:"id"`
	RevokeToken string `doc:"Secret capability returned at share creation." header:"X-Revoke-Token"`
}

// shareSettingsRespBody contains the share limits enforced by the server.
type shareSettingsRespBody struct {
	MaxEncryptedBytes uint  `doc:"Maximum encrypted share envelope size in bytes." json:"max_encrypted_bytes"`
	MaxTTLSeconds     int64 `doc:"Maximum share lifetime in seconds."              json:"max_ttl_seconds"`
	MaxViews          uint  `doc:"Maximum number of opens a share may allow."      json:"max_views"`
}

// shareSettingsResp is the public share settings response.
type shareSettingsResp struct {
	Body shareSettingsRespBody
}

// RegisterRoutes adds Share's HTTP operations to an API group.
func RegisterRoutes(api huma.API, share *Share, cfg config.ShareConfig) {
	huma.Get(api, "/settings", func(context.Context, *struct{}) (*shareSettingsResp, error) {
		return &shareSettingsResp{Body: shareSettingsRespBody{
			MaxEncryptedBytes: cfg.MaxEncryptedBytes,
			MaxTTLSeconds:     int64(cfg.MaxTTL / time.Second),
			MaxViews:          cfg.MaxViews,
		}}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "getShareSettings"
		o.Summary = "Get share settings"
	})

	huma.Post(api, "", func(ctx context.Context, input *createShareReq) (*createShareResp, error) {
		created, err := share.Create.Exec(ctx, CreateInput{
			EncryptedBlob: []byte(input.Body.Envelope),
			ExpiresIn:     time.Duration(input.Body.ExpiresInSeconds) * time.Second,
			Views:         input.Body.Views,
			CryptoVersion: 1,
		})
		if errors.Is(err, errbrick.ErrInvalidData) {
			return nil, huma.NewError(http.StatusBadRequest, err.Error())
		}
		if err != nil {
			slog.ErrorContext(ctx, "create share", "error", err)
			return nil, huma.NewError(http.StatusInternalServerError, "internal error")
		}

		return &createShareResp{Body: createShareRespBody(created)}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "createShare"
		o.Summary = "Create an encrypted share"
		o.DefaultStatus = http.StatusCreated
		o.MaxBodyBytes = int64(cfg.MaxEncryptedBytes) + maxJSONOverhead
		o.Errors = []int{http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusInternalServerError}
	})

	huma.Post(api, "/{id}/open", func(ctx context.Context, input *openShareReq) (*openShareResp, error) {
		record, err := share.Open.Exec(ctx, input.ID)
		if errors.Is(err, errbrick.ErrNotFound) {
			return nil, huma.NewError(http.StatusNotFound, "share is unavailable")
		}
		if err != nil {
			slog.ErrorContext(ctx, "open share", "error", err)
			return nil, huma.NewError(http.StatusInternalServerError, "internal error")
		}

		return &openShareResp{Body: openShareRespBody{
			Envelope:  string(record.EncryptedBlob),
			ViewsLeft: record.ViewsLeft,
			Burned:    record.ViewsLeft != nil && *record.ViewsLeft == 0,
		}}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "openShare"
		o.Summary = "Retrieve an encrypted share"
		o.Errors = []int{http.StatusNotFound, http.StatusInternalServerError}
	})

	huma.Delete(api, "/{id}", func(ctx context.Context, input *revokeShareReq) (*struct{}, error) {
		if input.RevokeToken == "" {
			return nil, huma.NewError(http.StatusNotFound, "share is unavailable")
		}
		if err := share.Revoke.Exec(ctx, input.ID, input.RevokeToken); err != nil {
			if errors.Is(err, errbrick.ErrNotFound) {
				return nil, huma.NewError(http.StatusNotFound, err.Error())
			}
			slog.ErrorContext(ctx, "revoke share", "error", err)
			return nil, huma.NewError(http.StatusInternalServerError, "internal error")
		}

		return &struct{}{}, nil
	}, func(o *huma.Operation) {
		o.OperationID = "revokeShare"
		o.Summary = "Revoke an encrypted share"
		o.Errors = []int{http.StatusNotFound, http.StatusInternalServerError}
	})
}
