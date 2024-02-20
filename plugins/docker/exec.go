package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/iBug/api-ustc/common"
)

type BaseConfig struct {
	Host      string        `json:"host"`
	Container string        `json:"container"`
	Timeout   time.Duration `json:"timeout"`
}

type Commander struct {
	docker    *client.Client
	container string
	timeout   time.Duration
}

func (c *Commander) Exec(cmd string) (string, error) {
	stream, err := c.docker.ContainerAttach(context.Background(), c.container, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	stream.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err = stream.Conn.Write([]byte("playing\n"))
	if err != nil {
		return "", fmt.Errorf("write to container %s: %w", c.container, err)
	}

	builder := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		stream.Conn.SetReadDeadline(time.Now().Add(c.timeout))
		n, err := stream.Reader.Read(buf)
		builder.Write(buf[:n])
		if errors.Is(err, os.ErrDeadlineExceeded) {
			break
		} else if err != nil {
			return "", fmt.Errorf("read from container %s: %w", c.container, err)
		}
	}
	return builder.String(), nil
}

func NewCommander(rawConfig json.RawMessage) (common.Commander, error) {
	config := BaseConfig{}
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	docker, _ := client.NewClientWithOpts(
		client.WithHost(config.Host),
		client.WithAPIVersionNegotiation(),
	)
	return &Commander{
		docker:    docker,
		container: config.Container,
		timeout:   config.Timeout,
	}, nil
}

func init() {
	common.Commanders.Register("docker.exec", NewCommander)
}
