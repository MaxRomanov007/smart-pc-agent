package executeScript

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/MaxRomanov007/smart-pc-go-lib/commands"
	"github.com/MaxRomanov007/smart-pc-go-lib/domain/models/message"
	"github.com/MaxRomanov007/smart-pc-go-lib/logger/sl"
	lua "github.com/yuin/gopher-lua"
	logLuaApi "smart-pc-agent/internal/commands/script-api/log"
	"smart-pc-agent/internal/storage/sqlite/dbqueries"
)

const (
	TypeBool   = 1
	TypeNumber = 2
	TypeString = 3
)

func New(log *slog.Logger, queries *dbqueries.Queries) commands.CommandFunc {
	return func(ctx context.Context, msg *message.Message) error {
		const op = "commands.handlers.execute-script"

		log := log.With(sl.Op(op), sl.MsgId(msg))

		script, err := queries.GetScriptById(ctx, msg.Payload.Data.Command)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Warn("script not found", slog.String("command", msg.Payload.Data.Command))
				return commands.Error("command not found")
			}
			return fmt.Errorf("%s: failed to get script: %w", op, err)
		}

		scriptParams, err := queries.GetScriptParams(ctx, script.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: failed to get script: %w", op, err)
		}

		messageParams, err := message.Parameter[map[string]string](msg)
		if err != nil {
			log.Warn(
				"failed to parse message parameters",
				slog.Any("parameter", msg.Payload.Data.Parameter),
				sl.Err(err),
			)
			return commands.Error("failed to get message parameters")
		}

		l := lua.NewState(lua.Options{
			SkipOpenLibs: true,
		})
		defer l.Close()

		paramsTable := createParamsTable(log, l, scriptParams, messageParams)
		l.SetGlobal("params", paramsTable)

		apiTable := createApiTable(l, log)
		l.SetGlobal("api", apiTable)

		if err := l.DoString(script.Text); err != nil {
			return fmt.Errorf("%s: failed to execute script: %w", op, err)
		}

		return nil
	}
}

func createParamsTable(
	logger *slog.Logger,
	l *lua.LState,
	scriptParams []*dbqueries.ScriptParam,
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

func createApiTable(l *lua.LState, log *slog.Logger) *lua.LTable {
	apiTable := l.NewTable()

	l.SetField(apiTable, "log", logLuaApi.New(l, log))

	return apiTable
}
