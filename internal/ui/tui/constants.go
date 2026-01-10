package tui

import "time"

// Logs page defaults.
const (
	logsMaxLines        = 5000
	logsDefaultTail     = 200
	logsRebuildInterval = 100 * time.Millisecond
)

// Layout defaults.
const (
	// logsNonBodyRows is the number of rows occupied by non-body elements
	// (e.g. header/footer/blank lines) in the logs page layout.
	//
	// Why:
	// - Keep layout math stable and readable (avoid scattered magic numbers).
	// - Logs page is a viewport layout and reserves a few fixed rows.
	logsNonBodyRows  = 4

	// tableNonBodyRows is the number of rows occupied by non-table elements
	// (e.g. title/footer/blank lines) in the table-based pages layout.
	//
	// Why:
	// - Table pages share a consistent layout.
	// - Centralizing the constant avoids drift between pages.
	tableNonBodyRows = 4
)
