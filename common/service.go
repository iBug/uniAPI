package common

import (
	"encoding/json"
	"io"
	"net/http"
)

type ServiceConfig struct {
	Service string          `json:"service"`
	Config  json.RawMessage `json:"config"`
}

type Service interface {
	http.Handler
}

type CommanderConfig struct {
	Commander string          `json:"commander"`
	Config    json.RawMessage `json:"config"`
}

type Commander interface {
	Exec(cmd string) (string, error)
}

type StreamerConfig struct {
	Streamer string          `json:"streamer"`
	Config   json.RawMessage `json:"config"`
}

type Streamer interface {
	Stream() (Stream, error)
}

type Stream interface {
	io.ReadWriteCloser
}

type Activator interface {
	Start() error
	Stop() error
}
