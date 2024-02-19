package server

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/iBug/api-ustc/common"
)

type ServiceSet map[string]common.ServiceConfig

type Server struct {
	services map[string]common.Service
}

func NewServer(serviceset ServiceSet) (*Server, error) {
	s := new(Server)
	s.services = make(map[string]common.Service)
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
		s.services[path.Clean(key)] = service
	}
	return nil
}

func (s *Server) Start() error {
	for _, service := range s.services {
		if activator, ok := service.(common.Activator); ok {
			if err := activator.Start(); err != nil {
				log.Printf("Failed to start service: %v", err)
			}
		}
	}
	return nil
}

func (s *Server) Stop() error {
	for _, service := range s.services {
		if activator, ok := service.(common.Activator); ok {
			if err := activator.Stop(); err != nil {
				log.Printf("Failed to stop service: %v", err)
			}
		}
	}
	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	service, ok := s.services[key]
	if !ok {
		http.NotFound(w, r)
		return
	}
	service.ServeHTTP(w, r)
}
