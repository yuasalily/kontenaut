package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type routerModel struct {
	containerUC *usecase.ContainerUsecase
	imageUC     *usecase.ImageUsecase

	width  int
	height int

	nav NavBar

	currentPageID pageID
	currentPage   Page
}

// compile-time interface check
var _ tea.Model = routerModel{}

func New(containerUC *usecase.ContainerUsecase, imageUC *usecase.ImageUsecase) tea.Model {
	p := newOverviewPage()
	return routerModel{
		containerUC:   containerUC,
		imageUC:       imageUC,
		nav:           NewNavBar(pageMetas()),
		currentPageID: pageOverview,
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
		if to, ok := m.nav.PageIDFromKey(msg.String()); ok {
			return m, func() tea.Msg { return navigateMsg{to: to} }
		}
	case navigateMsg:
		switch msg.to {
		case pageOverview:
			m.currentPageID = pageOverview
			m.currentPage = newOverviewPage()
			m.applyWindowSizeToCurrentPage()
			return m, m.currentPage.Init()

		case pageImages:
			m.currentPageID = pageImages
			m.currentPage = newImagesPage(m.imageUC)
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
	nav := m.nav.View(m.currentPageID)
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
	h := max(m.height-m.nav.Height(), 1)
	updated, _ := m.currentPage.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: h,
	})
	m.currentPage = updated
}
