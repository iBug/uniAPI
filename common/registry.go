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

type NewCommanderFunc = func(json.RawMessage) (Commander, error)

var commanders = make(map[string]NewCommanderFunc)

func RegisterCommander(name string, newFunc NewCommanderFunc) {
	commanders[name] = newFunc
}

func GetCommander(name string) (NewCommanderFunc, bool) {
	newFunc, ok := commanders[name]
	return newFunc, ok
}
