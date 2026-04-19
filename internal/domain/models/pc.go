package models

type Pc struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CanPowerOn  bool   `json:"canPowerOn"`

	Commands []Command `json:"commands,omitempty"`
}
