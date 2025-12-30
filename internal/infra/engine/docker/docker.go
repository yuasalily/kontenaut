package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
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

func (d *DockerEngine) ListImages(ctx context.Context) ([]engine.ImageSummary, error) {
	result, err := d.cli.ImageList(ctx, client.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}

	out := make([]engine.ImageSummary, 0, len(result.Items))
	for _, img := range result.Items {
		id := img.ID
		id = strings.TrimPrefix(id, "sha256:")
		if len(id) > 12 {
			id = id[:12]
		}

		repoTags := "<none>"
		if len(img.RepoTags) > 0 {
			first := img.RepoTags[0]
			if len(img.RepoTags) > 1 {
				repoTags = fmt.Sprintf("%s (+%d)", first, len(img.RepoTags)-1)
			} else {
				repoTags = first
			}
		}

		createdAt := ""
		if img.Created > 0 {
			createdAt = time.Unix(img.Created, 0).Format("2006-01-02")
		}

		out = append(out, engine.ImageSummary{
			ID:        id,
			RepoTags:  repoTags,
			Size:      formatBytes(img.Size),
			CreatedAt: createdAt,
		})
	}

	return out, nil
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

func formatBytes(n int64) string {
	if n < 0 {
		return "0 B"
	}
	return humanize.Bytes(uint64(n))
}
