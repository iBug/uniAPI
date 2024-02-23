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

type Attacher struct {
	docker    *client.Client
	container string
	timeout   time.Duration
}

func (c *Attacher) Execute(cmd string) (string, error) {
	ctx := context.Background()
	stream, err := c.docker.ContainerAttach(ctx, c.container, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
		Stderr: false,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()
	reader := demuxStream(stream.Reader, hasTty(c.docker, ctx, c.container), false)

	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}

	stream.Conn.SetWriteDeadline(time.Now().Add(c.timeout))
	_, err = stream.Conn.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("write to container %s: %w", c.container, err)
	}

	builder := new(strings.Builder)
	buf := make([]byte, 4096)
	for {
		stream.Conn.SetReadDeadline(time.Now().Add(c.timeout))
		n, err := reader.Read(buf)
		builder.Write(buf[:n])
		if errors.Is(err, os.ErrDeadlineExceeded) {
			break
		} else if err != nil {
			return "", fmt.Errorf("read from container %s: %w", c.container, err)
		}
	}
	return builder.String(), nil
}

func NewAttacher(rawConfig json.RawMessage) (*Attacher, error) {
	config := BaseConfig{}
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	docker, err := DockerClient(config)
	if err != nil {
		return nil, err
	}
	return &Attacher{
		docker:    docker,
		container: config.Container,
		timeout:   config.Timeout,
	}, nil
}

func NewAttacherStreamer(rawConfig json.RawMessage) (common.Streamer, error) {
	return NewAttacher(rawConfig)
}

func NewAttacherCommander(rawConfig json.RawMessage) (common.Commander, error) {
	return NewAttacher(rawConfig)
}

func init() {
	common.Streamers.Register("docker.stream", NewAttacherStreamer)
	common.Commanders.Register("docker.attachexec", NewAttacherCommander)
}
