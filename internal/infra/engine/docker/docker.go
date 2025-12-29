package docker

import (
	"context"
	"strings"

	"github.com/moby/moby/client"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

type DockerEngine struct {
	cli *client.Client
}

var _ engine.Engine = (*DockerEngine)(nil)

func New() (*DockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerEngine{cli: cli}, nil
}

func (d *DockerEngine) Close() error {
	return d.cli.Close()
}

func (d *DockerEngine) ListContainers(ctx context.Context) ([]engine.ContainerSummary, error) {
	result, err := d.cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	out := make([]engine.ContainerSummary, 0, len(result.Items))
	for _, c := range result.Items {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		out = append(out, engine.ContainerSummary{
			ID:     c.ID,
			Name:   name,
			Status: c.Status,
			Image:  c.Image,
		})
	}

	return out, nil
}
