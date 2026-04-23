package models

type CommandParameter struct {
	ID          string `json:"id,omitempty"`
	CommandID   string `json:"commandId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        int16  `json:"type"`

	Command *Command `json:"command,omitempty"`
}
