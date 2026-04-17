package log

import (
	"log/slog"

	lua "github.com/yuin/gopher-lua"
)

func New(l *lua.LState, log *slog.Logger) lua.LValue {
	table := l.NewTable()

	l.SetField(table, "debug", logDebug(l, log))
	l.SetField(table, "info", logInfo(l, log))
	l.SetField(table, "warn", logWarn(l, log))
	l.SetField(table, "error", logError(l, log))

	return table
}

func logDebug(l *lua.LState, log *slog.Logger) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		log.Debug(l.Get(-1).String())
		return 0
	})
}

func logInfo(l *lua.LState, log *slog.Logger) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		log.Info(l.Get(-1).String())
		return 0
	})
}

func logWarn(l *lua.LState, log *slog.Logger) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		log.Warn(l.Get(-1).String())
		return 0
	})
}

func logError(l *lua.LState, log *slog.Logger) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		log.Error(l.Get(-1).String())
		return 0
	})
}
