package commands

import (
	"time"
)

const (
	DoneCode = iota
	ScriptErrorCode
	InternalErrorCode
)

const (
	responseCommandLogType = "command-log"
)

type ResponseData struct {
	Command     string    `json:"command"`
	ReceivedAt  time.Time `json:"receivedAt"`
	CompletedAt time.Time `json:"completeAt"`
	Code        int       `json:"code"`
	Message     string    `json:"message,omitempty"`
}

type ResponsePayload struct {
	Type string       `json:"type"`
	Data ResponseData `json:"data"`
}

type Response struct {
	Retained bool            `json:"retained"`
	QOS      byte            `json:"qos"`
	Payload  ResponsePayload `json:"payload"`
}

func newResponse(command, message string, code int, receivedAt, completedAt time.Time) *Response {
	return &Response{
		Retained: false,
		QOS:      1,
		Payload: ResponsePayload{
			Type: responseCommandLogType,
			Data: ResponseData{
				Command:     command,
				ReceivedAt:  receivedAt,
				CompletedAt: completedAt,
				Message:     message,
				Code:        code,
			},
		},
	}
}

func Done(command string, receivedAt, completedAt time.Time) *Response {
	return newResponse(command, "", DoneCode, receivedAt, completedAt)
}

func ScriptFailed(command string, err *ScriptError, receivedAt, completedAt time.Time) *Response {
	return newResponse(command, err.Error(), ScriptErrorCode, receivedAt, completedAt)
}

func Internal(command string, receivedAt, completedAt time.Time) *Response {
	return newResponse(command, "internal error", InternalErrorCode, receivedAt, completedAt)
}
