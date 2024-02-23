package docker

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type BaseConfig struct {
	Host      string        `json:"host"`
	Container string        `json:"container"`
	Timeout   time.Duration `json:"timeout"`
}

func DockerClient(config BaseConfig) (*client.Client, error) {
	return client.NewClientWithOpts(
		client.WithHost(config.Host),
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(config.Timeout),
	)
}

func hasTty(cli *client.Client, ctx context.Context, container string) bool {
	info, err := cli.ContainerInspect(ctx, container)
	if err != nil {
		return false
	}
	return info.Config.Tty
}

func demuxStream(r io.Reader, tty bool, stderr bool) io.Reader {
	if tty {
		return r
	}
	pipeR, pipeW := io.Pipe()
	if stderr {
		go stdcopy.StdCopy(io.Discard, pipeW, r)
	} else {
		go stdcopy.StdCopy(pipeW, io.Discard, r)
	}
	return pipeR
}
