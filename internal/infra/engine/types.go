package engine

type DaemonInfo struct {
	ServerVersion    string
	OperationgSystem string
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
	Status  string
	Image   string
	ImageID string
}
