package tui

import "github.com/yuasalily/kontenaut/internal/infra/engine"

// Container action policy (minimal, initial).
// Why a dedicated policy file:
// - Containers have multiple action (start/stop/restart/delete/logs).
// - Keep action availability decisions in one place for consistency and future expansion.

func canStartContainer(state string) bool {
	// Minimal policy (initial):
	// - exited/created: start allowed
	// - running: start not allowed
	// - other states: keep conservative (not allowed)
	return state == "exited" || state == "created"
}

func canDeleteContainer(state string) bool {
	// Minimal policy (initial):
	// - running: delete not allowed (no force delete in this app)
	// - other states: delete allowed
	return state == "running"
}

// nonDeletableContainerIDs returns IDs of containers that cannot be deleted by policy.
func nonDeletableContainerIDs(items []engine.ContainerSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, c := range items {
		if !canDeleteContainer(c.State) {
			out[c.ID] = struct{}{}
		}
	}
	return out
}
