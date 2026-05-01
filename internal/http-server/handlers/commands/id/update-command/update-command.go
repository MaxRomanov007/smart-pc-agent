package updateCommand

import (
	"context"
	"errors"
	"go/types"
	"log/slog"
	"net/http"
	"smart-pc-agent/internal/http-server/middlewares/uuidmw/commands"
	"smart-pc-agent/internal/storage"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/MaxRomanov007/smart-pc-go-lib/middlewares/reqmw"
	"github.com/go-chi/render"
)

type RequestParameter struct {
	Name        string `json:"name"                  validate:"required,max=255"`
	Description string `json:"omitempty,description" validate:"omitempty,max=1024"`
	Type        int16  `json:"type"                  validate:"required,min=1,max=3"`
}

type Request struct {
	Name        string             `json:"name"                 validate:"omitempty,max=255"`
	Description string             `json:"description"          validate:"omitempty,max=1024"`
	Script      string             `json:"script"               validate:"omitempty,max=8192"`
	Parameters  []RequestParameter `json:"parameters,omitempty" validate:"omitempty,max=10,unique=Name,dive"`
}

type ServerCommandUpdater interface {
	UpdatePcCommand(ctx context.Context, command models.Command) (models.Command, error)
}

type LocalCommandUpdater interface {
	UpdateCommand(ctx context.Context, command models.Command) (models.Command, error)
}

func New(
	log *slog.Logger,
	localUpdater LocalCommandUpdater,
	serverUpdater ServerCommandUpdater,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.commands.get-commands"
		log := log.With(sl.Op(op), sl.ReqID(r))

		commandID := commands.MustCommandID(r)
		req := reqmw.MustGet[Request](r)

		parameters := make([]models.CommandParameter, len(req.Parameters))
		for i, p := range req.Parameters {
			parameters[i] = models.CommandParameter{
				Name:        p.Name,
				Description: p.Description,
				Type:        p.Type,
			}
		}

		command := models.Command{
			ID:          &commandID,
			Name:        req.Name,
			Description: req.Description,
			Script:      req.Script,
			Parameters:  parameters,
		}

		updatedCommand, err := localUpdater.UpdateCommand(r.Context(), command)
		if errors.Is(err, storage.ErrNotFound) {
			log.Warn("command not found")
			render.JSON(w, r, response.NotFound("command not found"))
			return
		}
		if err != nil {
			log.Error("failed to update command locally", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("command updated locally", slog.Any("command", updatedCommand))

		updatedCommand, err = serverUpdater.UpdatePcCommand(r.Context(), command)
		if err != nil {
			log.Error("failed to update command on server", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("command updated on server", slog.Any("command", updatedCommand))
		render.JSON(w, r, response.OK[types.Nil](nil))
		return
	}
}
