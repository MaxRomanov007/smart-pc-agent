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

func (m *Module) Register(l *lua.LState, table *lua.LTable) {
	l.SetField(table, "debug", m.logDebug(l))
	l.SetField(table, "info", m.logInfo(l))
	l.SetField(table, "warn", m.logWarn(l))
	l.SetField(table, "error", m.logError(l))
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
			"debug": {
				Description: "debug level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			"info": {
				Description: "info level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			"warn": {
				Description: "warn level logging",
				Params: []luaApi.ParamDoc{
					{
						Name:        "message",
						Type:        luaApi.TypeString,
						Description: "message text",
					},
				},
			},
			"error": {
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
