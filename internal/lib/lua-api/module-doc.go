package luaApi

const (
	TypeString  = "string"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeTable   = "table"
	TypeAny     = "any"
)

// ParamDoc описывает параметр функции
type ParamDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Optional    bool   `json:"optional,omitempty"`
}

// ReturnDoc описывает возвращаемое значение
type ReturnDoc struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// FunctionDoc описывает одну Lua-функцию
type FunctionDoc struct {
	Description string      `json:"description"`
	Params      []ParamDoc  `json:"params"`
	Returns     []ReturnDoc `json:"returns"`
	Example     string      `json:"example,omitempty"`
}

// ModuleDoc описывает весь модуль
type ModuleDoc struct {
	Description string                 `json:"description"`
	Functions   map[string]FunctionDoc `json:"functions"`
}
