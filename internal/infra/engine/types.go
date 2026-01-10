package engine

// DaemonInfo is a minimal set of daemon metadata shown in the Overview page.
type DaemonInfo struct {
	ServerVersion   string
	OperatingSystem string
}

// ImageSummary is a view-model friendly image summary for the Images page.
type ImageSummary struct {
	ID        string
	RepoTags  string
	Size      string
	CreatedAt string
}

// ContainerSummary is a view-model friendly container summary for the Containers page.
//
// State is a machine-readable value used for decisions such as "locked" (e.g. running containers).
type ContainerSummary struct {
	ID      string
	Name    string
	State   string // machine-readable state (e.g. "running", "exited")
	Status  string
	Image   string
	ImageID string
}
