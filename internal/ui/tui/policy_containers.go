package tui

import "github.com/yuasalily/kontenaut/internal/infra/engine"

// Container action policy (minimal, initial).
// Why a dedicated policy file:
// - Containers have multiple action (start/stop/restart/delete/logs).
// - Keep action availability decisions in one place for consistency and future expansion.

func canStopContainer(state string) bool {
	// Minimal policy (initial):
	// - running: stop allowed
	// - other states: conservative (not allowed)
	return state == "running"
}

func canRestartContainer(state string) bool {
	// Minimal policy (initial):
	// - running: restart allowed
	// - other states: conservative (not allowed)
	return state == "running"
}

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
	return state != "running"
}

// nonStartableContainerIDs returns IDs of containers that cannot be started by policy.
func nonStartableContainerIDs(items []engine.ContainerSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, c := range items {
		if !canStartContainer(c.State) {
			out[c.ID] = struct{}{}
		}
	}
	return out
}

// nonStoppableContainerIDs returns IDs of containers that cannot be stopped by policy.
func nonStoppableContainerIDs(items []engine.ContainerSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, c := range items {
		if !canStopContainer(c.State) {
			out[c.ID] = struct{}{}
		}
	}
	return out
}

// nonRestartableContainerIDs returns IDs of containers that cannot be restarted by policy.
func nonRestartableContainerIDs(items []engine.ContainerSummary) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, c := range items {
		if !canRestartContainer(c.State) {
			out[c.ID] = struct{}{}
		}
	}
	return out
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
