package docker

import (
	"encoding/json"

	"github.com/docker/docker/client"
	"github.com/iBug/api-ustc/common"
)

type BaseConfig struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type Commander struct {
	container string
	docker    *client.Client
}

func (c *Commander) Exec(cmd string) (string, error) {
	return "", nil
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
	return &Commander{container: config.Container, docker: docker}, nil
}

func init() {
	common.Commanders.Register("docker.exec", NewCommander)
}
