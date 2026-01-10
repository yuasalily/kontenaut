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
	logsNonBodyRows  = 4

	// tableNonBodyRows is the number of rows occupied by non-table elements
	// (e.g. title/footer/blank lines) in the table-based pages layout.
	tableNonBodyRows = 4
)
