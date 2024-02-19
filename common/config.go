package common

import (
	"os"
	"path/filepath"
)

func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config/api-ustc.yml"), nil
}
