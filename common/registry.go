package common

import "encoding/json"

type NewServiceFunc = func(json.RawMessage) (Service, error)

var services = make(map[string]NewServiceFunc)

func RegisterService(name string, newFunc NewServiceFunc) {
	services[name] = newFunc
}

func GetService(name string) (NewServiceFunc, bool) {
	newFunc, ok := services[name]
	return newFunc, ok
}
