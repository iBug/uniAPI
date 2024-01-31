package palworld

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	rcon "github.com/forewing/csgo-rcon"
	"github.com/iBug/api-ustc/common"
)

type Status struct {
	Players []string `json:"players"`
}

type Config struct {
	common.RconConfig
}

type Client struct {
	rcon *rcon.Client
}

func NewClient(config Config) *Client {
	c := &Client{
		rcon: common.RconClient(config.RconConfig),
	}
	c.rcon.SetCheckRequestID(false)
	return c
}

func (c *Client) GetStatus() (Status, error) {
	status := Status{}
	msg, err := c.rcon.Execute("ShowPlayers")
	if err != nil {
		return status, err
	}
	s := bufio.NewScanner(strings.NewReader(msg))
	s.Scan() // Discard first line
	for s.Scan() {
		line := s.Text()
		line = strings.Split(line, ",")[0]
		status.Players = append(status.Players, line)
	}
	return status, nil
}

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	status, err := c.GetStatus()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "internal server error"}`))
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=5")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}
