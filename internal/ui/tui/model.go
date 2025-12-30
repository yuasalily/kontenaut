package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type routerModel struct {
	containerUC *usecase.ContainerUsecase

	width  int
	height int

	currentPageID pageID
	currentPage   Page
}

// compile-time interface check
var _ tea.Model = routerModel{}

func New(containerUC *usecase.ContainerUsecase) tea.Model {
	p := newStartPage()
	return routerModel{
		containerUC:   containerUC,
		currentPageID: pageStart,
		currentPage:   p,
	}
}

func (m routerModel) Init() tea.Cmd {
	return m.currentPage.Init()
}

func (m routerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyWindowSizeToCurrentPage()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
		if to, ok := pageIDFromKey(msg.String()); ok {
			return m, func() tea.Msg { return navigateMsg{to: to} }
		}
	case navigateMsg:
		switch msg.to {
		case pageStart:
			m.currentPageID = pageStart
			m.currentPage = newStartPage()
			m.applyWindowSizeToCurrentPage()
			return m, m.currentPage.Init()

		case pageContainers:
			m.currentPageID = pageContainers
			m.currentPage = newContainersPage(m.containerUC)
			m.applyWindowSizeToCurrentPage()
			return m, m.currentPage.Init()
		}
	}

	updated, cmd := m.currentPage.Update(msg)
	m.currentPage = updated
	return m, cmd
}

func (m routerModel) View() string {
	nav := renderNavBar(m.currentPageID)
	page := m.currentPage.View()
	if nav == "" {
		return page
	}
	return lipgloss.JoinVertical(lipgloss.Left, nav, page)
}

func (m *routerModel) applyWindowSizeToCurrentPage() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	updated, _ := m.currentPage.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
	m.currentPage = updated
}

func pageIDFromKey(k string) (pageID, bool) {
	for _, meta := range pageMetas() {
		if meta.Key == k {
			return meta.ID, true
		}
	}
	return 0, false
}

var (
	navBaseStyle   = lipgloss.NewStyle().Padding(0, 1).Faint(true)
	navActiveStyle = lipgloss.NewStyle().Padding(0, 1).Bold(true)
)

func renderNavBar(current pageID) string {
	metas := pageMetas()
	if len(metas) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metas))
	for _, meta := range metas {
		label := "[" + meta.Key + "] " + meta.Title
		if meta.ID == current {
			parts = append(parts, navActiveStyle.Render(label))
		} else {
			parts = append(parts, navBaseStyle.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}
