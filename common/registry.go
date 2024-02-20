package common

import "encoding/json"

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

var (
	Services   = RegistryT[Service]{entries: make(map[string]NewFuncT[Service])}
	Commanders = RegistryT[Commander]{entries: make(map[string]NewFuncT[Commander])}
)
