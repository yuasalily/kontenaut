package tui

// truncText truncates a string to fit in width w.
//
// Note:
// - This is byte-based truncation (same behavior as the previous per-page helpers).
// - It is good enough for typical Docker IDs/names and keeps behavior stable.
func truncText(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return s[:w]
	}
	return s[:w-1] + "..."
}
