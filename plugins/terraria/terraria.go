package terraria

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

var (
	RePlayersConnected = *regexp.MustCompile(`^(\d+) players? connected\.$`)
	RePlayerInfo       = *regexp.MustCompile(`^(.*?)\s+\([0-9A-Fa-f:.]*\)$`)
)

type Status struct {
	Count   int       `json:"count"`
	Players []string  `json:"players"`
	Time    time.Time `json:"time"`
}

type Config struct {
	common.StreamerConfig
}

type Client struct {
	container string
	docker    *client.Client
}

func NewClient(config Config) *Client {
	docker, _ := client.NewClientWithOpts(
		client.WithHost(config.Host),
		client.WithAPIVersionNegotiation(),
	)
	return &Client{container: config.Container, docker: docker}
}

func (c *Client) GetStatus() (Status, error) {
	status := Status{Time: time.Now().Truncate(time.Second)}
	stream, err := c.docker.ContainerAttach(context.Background(), c.container, types.ContainerAttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return status, fmt.Errorf("attach container %s: %w", c.container, err)
	}
	defer stream.Close()

	_, err = stream.Conn.Write([]byte("playing\n"))
	if err != nil {
		return status, fmt.Errorf("write to container %s: %w", c.container, err)
	}

	line, err := stream.Reader.ReadString('\n')
	if err != nil {
		return status, fmt.Errorf("read echo: %w", err)
	}
	line = strings.TrimSpace(line)
	if line != "playing" {
		return status, fmt.Errorf("read echo: expected 'playing', got %#v", line)
	}

	var header [2]byte
	n, err := io.ReadFull(stream.Reader, header[:])
	if err != nil {
		return status, fmt.Errorf("read header: %w", err)
	}
	if n != 2 {
		return status, fmt.Errorf("read header: expected 2 bytes, got %d", n)
	}
	if string(header[:]) != ": " {
		return status, fmt.Errorf("read header: expected ': ', got %q", header)
	}

	for {
		line, err := stream.Reader.ReadString('\n')
		if err != nil {
			return status, fmt.Errorf("read line: %w", err)
		}
		line = strings.TrimSpace(line)

		if line == "No players connected." {
			break
		} else if line == "Invalid command." {
			return status, fmt.Errorf("invalid command")
		}

		m := RePlayersConnected.FindStringSubmatch(line)
		if len(m) == 2 {
			status.Count, _ = strconv.Atoi(m[1])
			break
		}
		m = RePlayerInfo.FindStringSubmatch(line)
		if len(m) == 2 {
			status.Players = append(status.Players, m[1])
		}
	}
	return status, nil
}

func (c *Client) Close() error {
	return c.docker.Close()
}

// ServeHTTP implements the http.Handler interface.
func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if true {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	status, err := c.GetStatus()
	if err != nil {
		log.Printf("Terraria error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
