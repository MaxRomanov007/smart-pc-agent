package luaApi

import (
	lua "github.com/yuin/gopher-lua"
)

// Module — интерфейс, который реализует каждый Lua-модуль
type Module interface {
	// Register регистрирует поля в lua-таблице
	Register(l *lua.LState, table *lua.LTable)
	// Doc возвращает документацию модуля
	Doc() ModuleDoc
}

type APISchema struct {
	Version string               `json:"version"`
	Modules map[string]ModuleDoc `json:"modules"`
}

type Registry struct {
	version string
	modules map[string]Module
}

func NewRegistry(version string) *Registry {
	return &Registry{
		version: version,
		modules: make(map[string]Module),
	}
}

// Register добавляет модуль под именем name (это имя поля в spc.*)
func (r *Registry) Register(name string, m Module) *Registry {
	r.modules[name] = m
	return r
}

// BuildTable собирает lua-таблицу spc для выполнения скрипта
func (r *Registry) BuildTable(l *lua.LState) *lua.LTable {
	spc := l.NewTable()
	for name, module := range r.modules {
		t := l.NewTable()
		module.Register(l, t)
		l.SetField(spc, name, t)
	}
	return spc
}

// Schema возвращает JSON-схему для фронта
func (r *Registry) Schema() APISchema {
	schema := APISchema{
		Version: r.version,
		Modules: make(map[string]ModuleDoc, len(r.modules)),
	}
	for name, module := range r.modules {
		schema.Modules[name] = module.Doc()
	}
	return schema
}
