package engine

type DaemonInfo struct {
	ServerVersion   string
	OperatingSystem string
}

type ImageSummary struct {
	ID        string
	RepoTags  string
	Size      string
	CreatedAt string
}

type ContainerSummary struct {
	ID      string
	Name    string
	State   string // machine-readable state (e.g. "running", "exited")
	Status  string
	Image   string
	ImageID string
}
