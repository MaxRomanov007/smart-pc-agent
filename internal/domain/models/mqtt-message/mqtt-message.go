package mqttMessage

import "github.com/eclipse/paho.golang/paho"

type Message[T any] struct {
	Type    string        `json:"type"`
	Data    T             `json:"data"`
	Publish *paho.Publish `json:"-"`
}
