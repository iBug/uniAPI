package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/iBug/api-ustc/common"
)

type Streamer struct {
	docker    *client.Client
	container string
}

type Stream struct {
	*types.HijackedResponse
}

func (s Stream) Read(p []byte) (n int, err error) {
	return s.Reader.Read(p)
}

func (s Stream) Write(p []byte) (n int, err error) {
	return s.Conn.Write(p)
}

func (s Stream) Close() error {
	return s.Conn.Close()
}

func (s *Streamer) Connect() (common.Stream, error) {
	stream, err := s.docker.ContainerAttach(context.Background(), s.container, container.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: true,
	})
	if err != nil {
		return Stream{}, err
	}
	return Stream{&stream}, nil
}

func NewStreamer(rawConfig json.RawMessage) (common.Streamer, error) {
	config := BaseConfig{}
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		return nil, err
	}

	docker, _ := client.NewClientWithOpts(
		client.WithHost(config.Host),
		client.WithAPIVersionNegotiation(),
	)
	return &Streamer{
		docker:    docker,
		container: config.Container,
	}, nil
}

func init() {
	common.Streamers.Register("docker.stream", NewStreamer)
}
