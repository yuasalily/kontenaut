package tui

import "github.com/charmbracelet/bubbles/table"

func colWidth(cols []table.Column, i int, fallback int) int {
	if i < 0 || i >= len(cols) {
		return fallback
	}
	return cols[i].Width
}
