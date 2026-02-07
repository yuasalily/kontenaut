package engine

import (
	"context"
	"io"
)

// Engine abstracts a container engine API used by usecases.
//
// Why:
// - Usecases must stay engine-agnostic (Docker/Podman/etc) and UI-agnostic.
// - Startup selects a single endpoint; the app does not switch engines at runtime.
type Engine interface {
	// Close releases underlying resources (e.g. SDK clients).
	Close() error

	// DaemonInfo returns basic daemon metadata shown in Overview.
	DaemonInfo(ctx context.Context) (DaemonInfo, error)

	// ListImages returns images for the Images page.
	ListImages(ctx context.Context) ([]ImageSummary, error)

	// ListContainers returns containers for the Containers page.
	ListContainers(ctx context.Context) ([]ContainerSummary, error)

	// StartContainer starts a container.
	//
	// Note:
	// This method returns when the daemon has accepted and completed the start operation.
	// It does NOT wait for the container's application to become "ready".
	StartContainer(ctx context.Context, containerID string) error

	// StopContainer stops a running container.
	//
	// Note:
	// This method returns when the daemon has accepted and completed the stop operation.
	// It does NOT wait for the container's application to become "ready".
	StopContainer(ctx context.Context, containerID string) error

	// RestartContainer restarts a running container.
	//
	// Note:
	// This method returns when the daemon has accepted and completed the restart operation.
	// It does NOT wait for the container's application to become "ready".
	RestartContainer(ctx context.Context, containerID string) error

	// RemoveImage removes an image.
	// Implementations may return an error when the image is in use.
	RemoveImage(ctx context.Context, imageID string) error

	// RemoveContainer removes a container.
	RemoveContainer(ctx context.Context, containerID string, force bool) error

	// ContainerLogs returns a snapshot of container logs without follow
	//
	// Note:
	// Follow-based log streaming is implemented separately
	// This snapshot API is kept intentionally for non-streaming use cases.
	ContainerLogs(ctx context.Context, containerID string, tail int) ([]string, error)

	// ContainerLogsFollow returns a streaming reader for container logs (tail + follow).
	// Why:
	// - Streaming is wrapped by usecases to hide io.ReadCloser from UI.
	// - Caller should cancel ctx to stop streaming promptly.
	// Caller must Close() it and should cancel the provided ctx to stop streaming.
	ContainerLogsFollow(ctx context.Context, containerID string, tail int) (io.ReadCloser, error)
}
