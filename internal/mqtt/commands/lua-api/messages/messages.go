package messages

import (
	luaApi "smart-pc-agent/internal/lib/lua-api"

	"github.com/ncruces/zenity"
	lua "github.com/yuin/gopher-lua"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

const (
	infoMessage  = "info"
	warnMessage  = "warn"
	errorMessage = "error"
)

func (m *Module) Register(l *lua.LState, table *lua.LTable) {
	l.SetField(table, infoMessage, m.infoDialog(l))
	l.SetField(table, warnMessage, m.warnDialog(l))
	l.SetField(table, errorMessage, m.ErrorDialog(l))
}

func (m *Module) infoDialog(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		_ = zenity.Info(
			l.Get(-1).String(),
			zenity.Title(l.Get(-2).String()),
			zenity.InfoIcon,
		)

		return 0
	})
}

func (m *Module) warnDialog(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		_ = zenity.Warning(
			l.Get(-1).String(),
			zenity.Title(l.Get(-2).String()),
			zenity.WarningIcon,
		)

		return 0
	})
}

func (m *Module) ErrorDialog(l *lua.LState) lua.LValue {
	return l.NewFunction(func(l *lua.LState) int {
		_ = zenity.Error(
			l.Get(-1).String(),
			zenity.Title(l.Get(-2).String()),
			zenity.ErrorIcon,
		)

		return 0
	})
}

func (m *Module) Doc() luaApi.ModuleDoc {
	return luaApi.ModuleDoc{
		Description: "message dialogs",
		Functions: map[string]luaApi.FunctionDoc{
			infoMessage: {
				Description: "info message dialog",
				Params: []luaApi.ParamDoc{
					{
						Name:        "title",
						Type:        luaApi.TypeString,
						Description: "dialog title",
					},
					{
						Name:        "text",
						Type:        luaApi.TypeString,
						Description: "dialog message text",
					},
				},
			},
			warnMessage: {
				Description: "warn message dialog",
				Params: []luaApi.ParamDoc{
					{
						Name:        "title",
						Type:        luaApi.TypeString,
						Description: "dialog title",
					},
					{
						Name:        "text",
						Type:        luaApi.TypeString,
						Description: "dialog message text",
					},
				},
			},
			errorMessage: {
				Description: "error message dialog",
				Params: []luaApi.ParamDoc{
					{
						Name:        "title",
						Type:        luaApi.TypeString,
						Description: "dialog title",
					},
					{
						Name:        "text",
						Type:        luaApi.TypeString,
						Description: "dialog message text",
					},
				},
			},
		},
	}
}
