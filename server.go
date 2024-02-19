package main

import (
	"fmt"
	"net/http"
	"path"

	"github.com/iBug/api-ustc/common"
)

type ServiceSet map[string]common.ServiceConfig

type Server struct {
	mux      http.ServeMux
	services []common.Service
}

func NewServer(serviceset ServiceSet) (*Server, error) {
	s := new(Server)
	err := s.loadServices(serviceset)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) loadServices(serviceset ServiceSet) error {
	for key, cfg := range serviceset {
		newFunc, ok := common.GetService(cfg.Service)
		if !ok {
			return fmt.Errorf("service %q not found", cfg.Service)
		}

		service, err := newFunc(cfg.Config)
		if err != nil {
			return fmt.Errorf("failed to create service %q: %v", cfg.Service, err)
		}
		s.services = append(s.services, service)

		urlPath := path.Clean("/" + key)
		s.mux.Handle(urlPath, service)
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
