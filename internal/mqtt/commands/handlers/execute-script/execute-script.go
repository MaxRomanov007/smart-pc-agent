package executeScript

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"smart-pc-agent/internal/domain/models"
	luaApi "smart-pc-agent/internal/lib/lua-api"
	"smart-pc-agent/internal/storage"
	"strconv"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models/message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	lua "github.com/yuin/gopher-lua"
)

const (
	TypeBool   = 1
	TypeNumber = 2
	TypeString = 3
)

type CommandGetter interface {
	GetCommandById(ctx context.Context, id string) (models.Command, error)
}

type CommandParamsGetter interface {
	GetCommandParams(ctx context.Context, commandId string) ([]models.CommandParameter, error)
}

func New(
	log *slog.Logger,
	commandGetter CommandGetter,
	paramsGetter CommandParamsGetter,
	registry *luaApi.Registry,
) commands.CommandFunc {
	return func(ctx context.Context, msg *message.Message) error {
		const op = "commands.handlers.execute-script"

		log := log.With(sl.Op(op), sl.MsgID(msg.Publish))

		command, err := commandGetter.GetCommandById(ctx, msg.Data.Command)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("script not found", slog.String("command", msg.Data.Command))
				return commands.Error("command not found")
			}
			return fmt.Errorf("%s: failed to get script: %w", op, err)
		}

		scriptParams, err := paramsGetter.GetCommandParams(ctx, command.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: failed to get script: %w", op, err)
		}

		messageParams, err := message.Parameter[map[string]string](msg)
		if err != nil {
			log.Warn(
				"failed to parse message parameters",
				slog.Any("parameter", msg.Data.Parameter),
				sl.Err(err),
			)
			return commands.Error("failed to get message parameters")
		}

		l := lua.NewState()
		defer l.Close()

		spc := registry.BuildTable(l)
		l.SetField(spc, "params", createParamsTable(log, l, scriptParams, messageParams))
		l.SetGlobal("spc", spc)

		err = l.DoString(command.Script)
		if apiErr, ok := errors.AsType[*lua.ApiError](err); ok {
			switch apiErr.Type {
			case lua.ApiErrorSyntax:
				return commands.Error("syntax error: " + apiErr.Error())
			case lua.ApiErrorFile:
				return commands.Error("file error: " + apiErr.Error())
			case lua.ApiErrorRun:
				return commands.Error("run error: " + apiErr.Error())
			case lua.ApiErrorError:
				return commands.Error("error: " + apiErr.Error())
			case lua.ApiErrorPanic:
				return commands.Error("panic error: " + apiErr.Error())
			default:
				return commands.Error("api error: " + apiErr.Error())
			}
		}
		if err != nil {
			return fmt.Errorf("%s: failed to execute script: %w", op, err)
		}

		return nil
	}
}

func createParamsTable(
	logger *slog.Logger,
	l *lua.LState,
	scriptParams []models.CommandParameter,
	messageParams map[string]string,
) *lua.LTable {
	const op = "commands.handlers.execute-script.createParamsTable"

	log := logger.With(sl.Op(op))

	paramsTable := l.NewTable()
	for _, param := range scriptParams {
		log := log.With(slog.String("parameterName", param.Name))

		stringParam, ok := messageParams[param.Name]
		if !ok {
			log.Info("parameter not found in message, setting to nil")
			l.SetField(paramsTable, param.Name, lua.LNil)
			continue
		}

		switch param.Type {
		case TypeBool:
			boolParam, err := strconv.ParseBool(stringParam)
			if err != nil {
				log.Warn(
					"failed to parse bool parameter, setting to nil",
					sl.Err(err),
				)
				l.SetField(paramsTable, param.Name, lua.LNil)
			}

			l.SetField(paramsTable, param.Name, lua.LBool(boolParam))

		case TypeNumber:
			numberParam, err := strconv.ParseFloat(stringParam, 64)
			if err != nil {
				log.Warn(
					"failed to parse number parameter, setting to nil",
					sl.Err(err),
				)
				l.SetField(paramsTable, param.Name, lua.LNil)
			}

			l.SetField(paramsTable, param.Name, lua.LNumber(numberParam))

		case TypeString:
			l.SetField(paramsTable, param.Name, lua.LString(stringParam))

		default:
			log.Warn("unknown type", slog.Any("parameter", param))
		}
	}

	return paramsTable
}
