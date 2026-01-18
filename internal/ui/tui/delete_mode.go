package tui

// deleteSelMark returns the selection mark used in delete-mode tables.
//
// Priority:
// - busy -> [*]
// - locked -> [#]
// - selected -> [x]
// - default -> [ ]
func deleteSelMark(id string, selected, locked, busy map[string]struct{}) string {
	if _, ok := busy[id]; ok {
		return "[*]"
	} else if _, ok := locked[id]; ok {
		return "[#]"
	} else if _, ok := selected[id]; ok {
		return "[x]"
	}
	return "[ ]"
}
