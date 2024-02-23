package common

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type TypeConfig struct {
	Type string `json:"type"`
}

type ServiceConfig struct {
	Service json.RawMessage `json:"service"`
}

type CommanderConfig struct {
	Commander json.RawMessage `json:"commander"`
}

type StreamerConfig struct {
	Streamer json.RawMessage `json:"streamer"`
}

func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config/api-ustc.yml"), nil
}
