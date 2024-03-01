package docker

import (
	"context"
	"encoding/json"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/iBug/uniAPI/common"
)

type LoggerConfig struct {
	BaseConfig
	Stderr bool `json:"stderr"`
}

type Logger struct {
	docker    *client.Client
	container string
	stderr    bool
}

type LogStream struct {
	r    io.Reader
	logs io.ReadCloser
}

func (s *LogStream) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

func (s *LogStream) Write(p []byte) (n int, err error) {
	return io.Discard.Write(p)
}

func (s *LogStream) Close() error {
	return s.logs.Close()
}

func (l *Logger) Connect() (common.Stream, error) {
	ctx := context.Background()
	options := container.LogsOptions{
		ShowStdout: !l.stderr,
		ShowStderr: l.stderr,
		Follow:     true,
		Tail:       "1",
	}
	logs, err := l.docker.ContainerLogs(ctx, l.container, options)
	if err != nil {
		return nil, err
	}

	r := demuxStream(logs, hasTty(l.docker, ctx, l.container), l.stderr)
	return &LogStream{r: r, logs: logs}, nil
}

func NewLogger(rawConfig json.RawMessage) (common.Streamer, error) {
	config := LoggerConfig{}
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, err
	}
	docker, err := DockerClient(config.BaseConfig)
	if err != nil {
		return nil, err
	}
	return &Logger{
		docker:    docker,
		container: config.Container,
		stderr:    config.Stderr,
	}, nil
}

func init() {
	common.Streamers.Register("docker.logs", NewLogger)
}
