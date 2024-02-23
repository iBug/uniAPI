package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/iBug/api-ustc/common"
)

type ServiceSet map[string]json.RawMessage

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
	var typeConfig common.TypeConfig
	for key, cfg := range serviceset {
		err := json.Unmarshal(cfg, &typeConfig)
		if err != nil {
			return fmt.Errorf("failed to parse service config: %v", err)
		}
		serviceType := typeConfig.Type
		newFunc, ok := common.Services.Get(serviceType)
		if !ok {
			return fmt.Errorf("service %q not found", serviceType)
		}

		service, err := newFunc(cfg)
		if err != nil {
			return fmt.Errorf("failed to create service %q: %v", serviceType, err)
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
