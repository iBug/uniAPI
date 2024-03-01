package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/iBug/uniAPI/common"
)

type Stream struct {
	*types.HijackedResponse
	r io.Reader
}

func (s Stream) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

func (s Stream) Write(p []byte) (n int, err error) {
	return s.Conn.Write(p)
}

func (s Stream) Close() error {
	return s.Conn.Close()
}

func (c *Attacher) Connect() (common.Stream, error) {
	ctx := context.Background()
	stream, err := c.docker.ContainerAttach(ctx, c.container, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return Stream{}, err
	}
	r := demuxStream(stream.Reader, hasTty(c.docker, ctx, c.container), false)
	return Stream{&stream, r}, nil
}
