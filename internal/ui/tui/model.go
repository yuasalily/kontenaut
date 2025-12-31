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

	dialog *dialogModel
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
	// ウィンドウサイズの処理
	if handled, cmd := m.handleWindowSize(msg); handled {
		return m, cmd
	}

	// ダイアログの処理
	if handled, cmd := m.handleDialog(msg); handled {
		return m, cmd
	}
	// ページの処理
	return m.updateNormal(msg)
}

func (m routerModel) View() string {
	if v, ok := m.dialogView(); ok {
		return v
	}
	return m.normalView()
}

func (m *routerModel) handleDialog(msg tea.Msg) (bool, tea.Cmd) {
	if m.dialog == nil {
		return false, nil
	}

	closed := m.dialog.Update(msg)
	if closed {
		m.dialog = nil
	}
	return true, nil
}

func (m *routerModel) handleWindowSize(msg tea.Msg) (bool, tea.Cmd) {
	ws, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return false, nil
	}

	m.width, m.height = ws.Width, ws.Height
	m.applyWindowSizeToCurrentPage()
	if m.dialog != nil {
		_ = m.dialog.Update(ws)
	}
	return true, nil
}

func (m routerModel) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	case imagesLoadFailedMsg:
		return m, showDialogCmd(dialogError, "Images", msg.err.Error())
	case containersLoadFailedMsg:
		return m, showDialogCmd(dialogError, "Containers", msg.err.Error())

	case showDialogMsg:
		m.dialog = newDialog(msg.kind, msg.title, msg.body)
		m.applyWindowSizeToDialog()
		return m, nil
	}

	updated, cmd := m.currentPage.Update(msg)
	m.currentPage = updated
	return m, cmd
}

func (m routerModel) dialogView() (string, bool) {
	if m.dialog == nil {
		return "", false
	}
	return m.dialog.View(), true
}

func (m routerModel) normalView() string {
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

func (m *routerModel) applyWindowSizeToDialog() {
	if m.dialog == nil {
		return
	}
	if m.width <= 0 || m.height <= 0 {
		return
	}
	_ = m.dialog.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
}
