package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
)

// imagesModeSpec centralized the "UI shape" per mode:
//
// Why:
// - Avoids mode/spec duplication (single source of truth for rendering)
// - Keeps modes "pure-ish" (input -> action) and free from router-owned mutable state.
type imagesModeSpec struct {
	Title      func() string
	Columns    func(totalWidth int) []table.Column
	FooterKeys func(ctx imagesCtx) []key.Binding
	Rows       func(items []engine.ImageSummary, cols []table.Column, selected, locked, busy map[string]struct{}) []table.Row
}

func imagesModeSpecs() map[imagesModeID]imagesModeSpec {
	return map[imagesModeID]imagesModeSpec{
		imagesModeNormal: imagesModeSpecNormal(),
		imagesModeDelete: imagesModeSpecDelete(),
	}
}

func imagesModeSpecNormal() imagesModeSpec {
	return imagesModeSpec{
		Title: func() string { return "Images" },
		Columns: func(totalWidth int) []table.Column {
			const (
				idW      = 12
				sizeW    = 10
				createdW = 12
			)

			repoW := 24
			if totalWidth > 0 {
				rest := totalWidth - (idW + sizeW + createdW) - 6
				if rest > repoW {
					repoW = rest
				}
			}

			return []table.Column{
				{Title: "ID", Width: idW},
				{Title: "REPO:TAG", Width: repoW},
				{Title: "SIZE", Width: sizeW},
				{Title: "CREATED", Width: createdW},
			}
		},
		FooterKeys: func(ctx imagesCtx) []key.Binding {
			return []key.Binding{
				ctx.km.DeleteSingle,
				ctx.km.EnterDeleteMode,
				ctx.km.Refresh,
				ctx.gkm.Quit,
			}
		},
		Rows: func(items []engine.ImageSummary, cols []table.Column, selected, locked, busy map[string]struct{}) []table.Row {
			return rowsFromImageSummariesNormal(items, cols)
		},
	}
}

func imagesModeSpecDelete() imagesModeSpec {
	return imagesModeSpec{
		Title: func() string { return lipgloss.NewStyle().Bold(true).Render("Images [DELETE MODE]") },
		Columns: func(totalWidth int) []table.Column {
			const (
				selW     = 4
				idW      = 12
				sizeW    = 10
				createdW = 12
			)

			repoW := 24
			if totalWidth > 0 {
				rest := totalWidth - (selW + idW + sizeW + createdW) - 8
				if rest > repoW {
					repoW = rest
				}
			}

			return []table.Column{
				{Title: "SEL", Width: selW},
				{Title: "ID", Width: idW},
				{Title: "REPO:TAG", Width: repoW},
				{Title: "SIZE", Width: sizeW},
				{Title: "CREATED", Width: createdW},
			}
		},
		FooterKeys: func(ctx imagesCtx) []key.Binding {
			return []key.Binding{
				ctx.km.Select,
				ctx.km.Execute,
				ctx.km.Exit,
				ctx.km.Refresh,
				ctx.gkm.Quit,
			}
		},
		Rows: func(items []engine.ImageSummary, cols []table.Column, selected, locked, busy map[string]struct{}) []table.Row {
			return rowsFromImageSummariesDelete(items, cols, selected, locked, busy)
		},
	}
}
