package common

import (
	"fmt"
	"log"
	"time"

	rcon "github.com/forewing/csgo-rcon"
)

type RconConfig struct {
	ServerAddr string `json:"server"`
	ServerPort int    `json:"port"`
	Password   string `json:"password"`
	Timeout    string `json:"timeout"`
}

func ParseDurationDefault(s string, def time.Duration) (d time.Duration) {
	d = time.Second
	if s != "" {
		t, err := time.ParseDuration(s)
		if err != nil {
			log.Printf("Invalid timeout %s, using %s\n", s, d)
		} else {
			d = t
		}
	}
	return
}

func RconClient(config RconConfig) *rcon.Client {
	return rcon.New(
		fmt.Sprintf("%s:%d", config.ServerAddr, config.ServerPort),
		config.Password,
		ParseDurationDefault(config.Timeout, 1*time.Second),
	)
}
