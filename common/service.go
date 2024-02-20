package common

import (
	"encoding/json"
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

type Activator interface {
	Start() error
	Stop() error
}
