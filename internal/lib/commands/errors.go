package commands

type ScriptError struct {
	Message string
}

func (e *ScriptError) Error() string {
	return e.Message
}
