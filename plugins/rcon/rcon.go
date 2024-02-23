package common

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/iBug/api-ustc/common"
	"github.com/iBug/api-ustc/plugins/rcon/internal/rcon"
)

type Config struct {
	ServerAddr string `json:"server"`
	ServerPort int    `json:"port"`
	Password   string `json:"password"`
	Timeout    string `json:"timeout"`
}

func ParseDurationDefault(s string, def time.Duration) time.Duration {
	if s == "" {
		return def
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("Invalid timeout %s, using %s\n", s, def)
		return def
	}
	return dur
}

func RconClient(config Config) *rcon.Client {
	return rcon.New(
		fmt.Sprintf("%s:%d", config.ServerAddr, config.ServerPort),
		config.Password,
		ParseDurationDefault(config.Timeout, 1*time.Second),
	)
}

func NewCommander(rawConfig json.RawMessage) (common.Commander, error) {
	config := Config{}
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, err
	}
	return RconClient(config), nil
}

func init() {
	common.Commanders.Register("rcon", NewCommander)
}
