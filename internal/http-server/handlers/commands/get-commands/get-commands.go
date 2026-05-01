package getCommands

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/MaxRomanov007/smart-pc-go-lib/api/response"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type CommandGetter interface {
	GetCommands(ctx context.Context) ([]models.Command, error)
}

type CommandParametersGetter interface {
	GetCommandParameters(ctx context.Context, id uuid.UUID) ([]models.CommandParameter, error)
}

type CommandScriptGetter interface {
	GetCommandScript(ctx context.Context, id uuid.UUID) (string, error)
}

func New(
	log *slog.Logger,
	commandGetter CommandGetter,
	commandParametersGetter CommandParametersGetter,
	commandScriptGetter CommandScriptGetter,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "http-server.handlers.commands.get-commands"
		log := log.With(sl.Op(op), sl.ReqID(r))

		commands, err := commandGetter.GetCommands(r.Context())
		if err != nil {
			log.Error("failed to get commands", sl.Err(err))
			render.JSON(w, r, response.InternalError())
			return
		}

		for i := 0; i < len(commands); i++ {
			log := log.With(slog.String("command_id", commands[i].ID.String()))

			commands[i].Parameters, err = commandParametersGetter.GetCommandParameters(
				r.Context(),
				*commands[i].ID,
			)
			if err != nil {
				log.Warn("failed to get command parameters", sl.Err(err))
			}

			commands[i].Script, err = commandScriptGetter.GetCommandScript(
				r.Context(),
				*commands[i].ID,
			)
			if err != nil {
				log.Warn("failed to get command script", sl.Err(err))
			}
		}

		render.JSON(w, r, response.OK(&commands))
	}
}
