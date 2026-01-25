package tui

// selMark returns the selection mark used in multi-select action-mode tables.
//
// Priority:
// - busy -> [*]
// - blocked -> [#]
// - selected -> [x]
// - default -> [ ]
//
// Note:
// - "blocked" means "action is not allowed by policy" (e.g. non-deletable, non-startable).
func selMark(id string, selected, blocked, busy map[string]struct{}) string {
	if _, ok := busy[id]; ok {
		return "[*]"
	} else if _, ok := blocked[id]; ok {
		return "[#]"
	} else if _, ok := selected[id]; ok {
		return "[x]"
	}
	return "[ ]"
}
