package models

type Command struct {
	ID          string `json:"id"`
	PcID        string `json:"pcId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Script      string `json:"script"`

	Parameters []CommandParameter `json:"parameters,omitempty"`
}
