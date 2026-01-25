package tui

// deleteSelMark returns the selection mark used in delete-mode tables.
//
// Priority:
// - busy -> [*]
// - nonDeletable -> [#]
// - selected -> [x]
// - default -> [ ]
func deleteSelMark(id string, selected, nonDeletable, busy map[string]struct{}) string {
	if _, ok := busy[id]; ok {
		return "[*]"
	} else if _, ok := nonDeletable[id]; ok {
		return "[#]"
	} else if _, ok := selected[id]; ok {
		return "[x]"
	}
	return "[ ]"
}
