package deleteCommand

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"smart-pc-agent/internal/domain/models"
	"smart-pc-agent/internal/services"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type LocalCommandDeleter interface {
	DeleteCommand(ctx context.Context, id string) (models.Command, error)
}

type ServerCommandDeleter interface {
	DeletePcCommand(ctx context.Context, id string) (models.Command, error)
}

type LocalCommandCreator interface {
	CreateCommand(ctx context.Context, command models.Command) (models.Command, error)
}

func New(
	log *slog.Logger,
	localDeleter LocalCommandDeleter,
	serverDeleter ServerCommandDeleter,
	localCreator LocalCommandCreator,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.commands.get-commands"
		log := log.With(sl.Op(op), sl.ReqID(r))

		commandID := chi.URLParam(r, "command_id")
		if commandID == "" {
			log.Warn("missing command id")
			render.JSON(w, r, response.BadRequest("missing command id"))
			return
		}

		deleted, err := localDeleter.DeleteCommand(r.Context(), commandID)
		if errors.Is(err, storage.ErrNotFound) {
			log.Warn("command not found", sl.Err(err))
			render.JSON(w, r, response.NotFound("command not found"))
			return
		}
		if err != nil {
			log.Error("failed to delete local command", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("local command deleted", slog.Any("deleted", deleted))

		_, err = serverDeleter.DeletePcCommand(r.Context(), commandID)
		if err == nil || errors.Is(err, services.ErrNotFound) {
			log.Debug("command deleted from server")
			render.JSON(w, r, response.OK(&deleted))
			return
		}

		log.Error("failed to delete command from server, rollback", sl.Err(err))

		_, err = localCreator.CreateCommand(r.Context(), deleted)
		if err != nil {
			log.Error("failed to create local command", sl.Err(err))
		}

		render.JSON(w, r, response.InternalError())
		return
	}
}
