package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type screen int

const (
	screenStart screen = iota
	screenContainers
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
		updated, cmd := m.currentPage.Update(msg)
		m.currentPage = updated
		return m, cmd
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
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
	return m.currentPage.View()
}

func (m *routerModel) applyWindowSizeToCurrentPage() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	updated, _ := m.currentPage.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
	m.currentPage = updated
}
