package deleteCommand

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"smart-pc-agent/internal/http-server/middlewares/uuidmw/commands"
	"smart-pc-agent/internal/services"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type LocalCommandDeleter interface {
	DeleteCommand(ctx context.Context, id uuid.UUID) (models.Command, error)
}

type ServerCommandDeleter interface {
	DeletePcCommand(ctx context.Context, id uuid.UUID) (models.Command, error)
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

		commandID := commands.MustCommandID(r)

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
