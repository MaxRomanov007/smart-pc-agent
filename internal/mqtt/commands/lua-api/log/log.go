package log

import (
	"log/slog"
	luaApi "smart-pc-agent/internal/lib/lua-api"

	lua "github.com/yuin/gopher-lua"
)

type Module struct {
	log *slog.Logger
}

func New(log *slog.Logger) *Module {
	return &Module{log: log}
}

const (
	debugKey = "debug"
	infoKey  = "info"
	warnKey  = "warn"
	errorKey = "error"
)

func (m *Module) Register(l *lua.LState, table *lua.LTable) {
	l.SetField(table, debugKey, m.logDebug(l))
	l.SetField(table, infoKey, m.logInfo(l))
	l.SetField(table, warnKey, m.logWarn(l))
	l.SetField(table, errorKey, m.logError(l))
}

func (m *Module) logDebug(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		m.log.Debug(l.Get(-1).String())
		return 0
	})
}

func (m *Module) logInfo(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		m.log.Info(l.Get(-1).String())
		return 0
	})
}

func (m *Module) logWarn(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		m.log.Warn(l.Get(-1).String())
		return 0
	})
}

func (m *Module) logError(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		m.log.Error(l.Get(-1).String())
		return 0
	})
}

func (m *Module) Doc() luaApi.ModuleDoc {
	return luaApi.ModuleDoc{
		Description: "logging",
		Functions: map[string]luaApi.FunctionDoc{
			debugKey: {
				Description: "debug level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			infoKey: {
				Description: "info level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			warnKey: {
				Description: "warn level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			errorKey: {
				Description: "error level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
		},
	}
}
