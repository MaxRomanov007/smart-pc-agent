package deleteThisPc

import (
	"context"
	"go/types"
	"log/slog"
	"net/http"
	"smart-pc-agent/internal/domain/models"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
)

type LocalPcDeleter interface {
	DeleteThisPc(ctx context.Context) error
}

type ServerPcDeleter interface {
	DeleteThisPc(ctx context.Context) (models.Pc, error)
}

type DbCleaner interface {
	CleanDb(ctx context.Context) error
}

func New(
	log *slog.Logger,
	localDeleter LocalPcDeleter,
	serverDeleter ServerPcDeleter,
	dbCleaner DbCleaner,
	stopApp func(),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.pc-id"
		log := log.With(sl.Op(op), sl.ReqID(r))

		if err := localDeleter.DeleteThisPc(r.Context()); err != nil {
			log.Error("failed to delete pc locally", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		if _, err := serverDeleter.DeleteThisPc(r.Context()); err != nil {
			log.Error("failed to delete pc on server", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("successfully deleted pc")
		render.JSON(w, r, response.OK[types.Nil](nil))

		if err := dbCleaner.CleanDb(r.Context()); err != nil {
			log.Error("failed to clean db", sl.Err(err))
		}

		log.Info("stopping application")
		stopApp()
		return
	}
}
