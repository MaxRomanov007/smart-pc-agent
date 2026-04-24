package pcId

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
)

type PcIDGetter interface {
	GetPcID(ctx context.Context) (string, error)
}

func New(
	log *slog.Logger,
	getter PcIDGetter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.pc-id"
		log := log.With(sl.Op(op), sl.ReqID(r))

		pcID, err := getter.GetPcID(r.Context())
		if errors.Is(err, storage.ErrNotFound) {
			log.Warn("pc id not found")
			render.JSON(w, r, response.NotFound("pc id not found"))
			return
		}
		if err != nil {
			log.Error("failed to get pc id", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("got pc id", slog.String("pc_id", pcID))
		render.JSON(w, r, response.OK(&pcID))
		return
	}
}
