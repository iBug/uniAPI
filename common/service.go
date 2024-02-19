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

type Activator interface {
	Start() error
	Stop() error
}
