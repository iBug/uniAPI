package factorio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	rcon "github.com/forewing/csgo-rcon"
	"github.com/iBug/api-ustc/common"
)

type Status struct {
	Time    time.Time `json:"time"`
	Players []string  `json:"players"`
}

type Config struct {
	common.RconConfig
}

type Client struct {
	ServerAddr string
	ServerPort int
	Password   string

	rcon *rcon.Client
}

func NewClient(serverAddr string, serverPort int, password string, timeout time.Duration) *Client {
	c := &Client{
		ServerAddr: serverAddr,
		ServerPort: serverPort,
		Password:   password,
	}
	c.rcon = rcon.New(fmt.Sprintf("%s:%d", c.ServerAddr, c.ServerPort), c.Password, timeout)
	return c
}

func (c *Client) GetStatus() (status Status, err error) {
	response, err := c.rcon.Execute("/players online")
	if err != nil {
		return
	}
	s := bufio.NewScanner(strings.NewReader(response))
	s.Scan()
	for s.Scan() {
		text := s.Text()
		if !strings.HasPrefix(text, "  ") {
			continue
		}
		text = strings.TrimPrefix(text, "  ")
		text = strings.TrimSuffix(text, " (online)")
		status.Players = append(status.Players, text)
	}
	status.Time = time.Now().Truncate(time.Second)
	return
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
