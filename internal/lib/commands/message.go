package commands

import (
	"encoding/json"
	"fmt"
)

type Data struct {
	Command   string          `json:"command"`
	Parameter json.RawMessage `json:"parameter"`
}

type Payload struct {
	Type string `json:"type"`
	Data Data   `json:"data"`
}

type Message struct {
	Duplicate bool    `json:"duplicate"`
	Qos       byte    `json:"qos"`
	Retained  bool    `json:"retained"`
	Topic     string  `json:"topic"`
	MessageID uint16  `json:"message_id"`
	Payload   Payload `json:"payload"`
}

func Parameter[T any](msg *Message) (T, error) {
	const op = "lib.commands.message.Parameter"

	var result T
	if err := json.Unmarshal(msg.Payload.Data.Parameter, &result); err != nil {
		return result, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return result, nil
}
