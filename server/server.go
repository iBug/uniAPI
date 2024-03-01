package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/iBug/uniAPI/common"
)

type ServiceSet map[string]json.RawMessage

type ServerConfig struct {
	Services ServiceSet `json:"services"`
}

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
	// var typeConfig common.TypeConfig
	for key, cfg := range serviceset {
		service, err := common.Services.NewFromConfig(cfg)
		if err != nil {
			return fmt.Errorf("failed to create service %q: %v", key, err)
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
	log.Printf("Serving %s", r.URL.Path)
	key := strings.SplitN(path.Clean(r.URL.Path), "/", 3)[1]
	service, ok := s.services[key]
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.StripPrefix("/"+key, service).ServeHTTP(w, r)
}

func NewServerFromConfig(rawConfig json.RawMessage) (common.Service, error) {
	var config ServerConfig
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}
	return NewServer(config.Services)
}

func init() {
	common.Services.Register("server", NewServerFromConfig)
}
