package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/infra/engine"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type imagesPage struct {
	imageUC *usecase.ImageUsecase

	loading bool
	images  []engine.ImageSummary
	err     error

	width  int
	height int

	imagesTable table.Model
}

// compile-time interface check
var _ Page = imagesPage{}

func newImagesPage(imageUC *usecase.ImageUsecase) Page {
	return imagesPage{
		imageUC:     imageUC,
		loading:     true,
		imagesTable: newImagesTable(),
	}
}

func (p imagesPage) Init() tea.Cmd { return listImagesCmd(p.imageUC) }

type imagesLoadedMsg []engine.ImageSummary
type imagesLoadFailedMsg struct{ err error }

func listImagesCmd(imageUC *usecase.ImageUsecase) tea.Cmd {
	return func() tea.Msg {
		items, err := imageUC.List(context.Background())
		if err != nil {
			return imagesLoadFailedMsg{err: err}
		}
		return imagesLoadedMsg(items)
	}
}

func (p imagesPage) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width, p.height = msg.Width, msg.Height
		p = p.applyTableLayout()
		return p, nil

	case imagesLoadedMsg:
		p.loading = false
		p.images = []engine.ImageSummary(msg)
		p.imagesTable.SetRows(rowsFromImageSummaries(p.images, p.imagesTable.Columns()))
		return p, nil

	case imagesLoadFailedMsg:
		p.loading = false
		p.err = msg.err
		return p, nil
	}
	var cmd tea.Cmd
	p.imagesTable, cmd = p.imagesTable.Update(msg)
	return p, cmd
}

func (p imagesPage) View() string {
	if p.loading {
		return "Loading..."
	}
	var b strings.Builder
	b.WriteString("Images\n")

	b.WriteString(p.imagesTable.View())
	b.WriteString("\n(q to quit)\n")
	return b.String()
}

func (p imagesPage) applyTableLayout() imagesPage {
	if p.width <= 0 || p.height <= 0 {
		return p
	}
	tableHeight := max(p.height-4, 1)

	p.imagesTable.SetWidth(p.width)
	p.imagesTable.SetHeight(tableHeight)

	cols := columnsForImagesWidth(p.width)
	p.imagesTable.SetColumns(cols)
	if len(p.images) > 0 {
		p.imagesTable.SetRows(rowsFromImageSummaries(p.images, cols))
	}
	return p
}

func newImagesTable() table.Model {
	cols := columnsForImagesWidth(0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(nil),
		table.WithFocused(true),
	)
	return t
}

func columnsForImagesWidth(total int) []table.Column {
	const (
		idW      = 12
		sizeW    = 10
		createdW = 12
	)

	repoW := 24
	if total > 0 {
		rest := total - (idW + sizeW + createdW) - 6
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
}

func rowsFromImageSummaries(items []engine.ImageSummary, cols []table.Column) []table.Row {
	getW := func(i int, fallback int) int {
		if i < 0 || i >= len(cols) {
			return fallback
		}
		return cols[i].Width
	}

	idW := getW(0, 12)
	repoW := getW(1, 24)
	sizeW := getW(2, 10)
	createdW := getW(3, 12)

	out := make([]table.Row, 0, len(items))
	for _, img := range items {
		row := table.Row{
			truncImage(img.ID, idW),
			truncImage(img.RepoTags, repoW),
			truncImage(img.Size, sizeW),
			truncImage(img.CreatedAt, createdW),
		}
		out = append(out, row)
	}
	return out
}

func truncImage(s string, w int) string {
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
