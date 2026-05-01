package createCommand

import (
	"context"
	"go/types"
	"log/slog"
	"net/http"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/MaxRomanov007/smart-pc-go-lib/middlewares/reqmw"
	"github.com/go-chi/render"
	"github.com/google/uuid"
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

type CommandServerSaver interface {
	CreatePcCommand(ctx context.Context, command models.Command) (models.Command, error)
}

type CommandServerDeleter interface {
	DeletePcCommand(ctx context.Context, id uuid.UUID) (models.Command, error)
}

type CommandLocalSaver interface {
	CreateCommand(ctx context.Context, command models.Command) (models.Command, error)
}

func New(
	log *slog.Logger,
	serverSaver CommandServerSaver,
	serverDeleter CommandServerDeleter,
	localSaver CommandLocalSaver,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.commands.create-command"
		log := log.With(sl.Op(op), sl.ReqID(r))

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
			Name:        req.Name,
			Description: req.Description,
			Script:      req.Script,
			Parameters:  parameters,
		}

		serverCommand, err := serverSaver.CreatePcCommand(r.Context(), command)
		if err != nil {
			log.Error("failed to save command on server", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		command.ID = serverCommand.ID

		log.Debug("saved command on server", slog.Any("command", serverCommand))

		_, err = localSaver.CreateCommand(r.Context(), command)
		if err != nil {
			if _, deleteErr := serverDeleter.DeletePcCommand(
				r.Context(),
				*command.ID,
			); deleteErr != nil {
				log.Error(
					"failed to delete command from server after local save failed",
					sl.Err(deleteErr),
				)
			}

			log.Error("failed to local save command", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		log.Debug("command saved locally")
		render.JSON(w, r, response.OK[types.Nil](nil))
		return
	}
}
