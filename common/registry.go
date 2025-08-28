package common

import (
	"encoding/json"
	"fmt"
)

type NewFuncT[T any] func(json.RawMessage) (T, error)

type RegistryT[T any] struct {
	entries map[string]NewFuncT[T]
}

func (r *RegistryT[T]) Register(name string, newFunc NewFuncT[T]) {
	r.entries[name] = newFunc
}

func (r *RegistryT[T]) Get(name string) (NewFuncT[T], bool) {
	newFunc, ok := r.entries[name]
	return newFunc, ok
}

func (r *RegistryT[T]) New(name string, b json.RawMessage) (T, error) {
	newFunc, ok := r.Get(name)
	if !ok {
		return *new(T), fmt.Errorf("%q not found", name)
	}
	return newFunc(b)
}

func (r *RegistryT[T]) NewFromConfig(b json.RawMessage) (T, error) {
	var config TypeConfig
	err := json.Unmarshal(b, &config)
	if err != nil {
		return *new(T), err
	}
	return r.New(config.Type, b)
}

var (
	Services   = RegistryT[Service]{entries: make(map[string]NewFuncT[Service])}
	Commanders = RegistryT[Commander]{entries: make(map[string]NewFuncT[Commander])}
	Streamers  = RegistryT[Streamer]{entries: make(map[string]NewFuncT[Streamer])}
)

// Convenience functions
func NewService[T Service](config json.RawMessage) (Service, error) {
	var s T
	err := json.Unmarshal(config, &s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
