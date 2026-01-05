package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

type modalSession struct {
	dialog    *dialogModel
	confirmID ConfirmID // confirm dialog出ない場合は空
}

func (s modalSession) isConfirm() bool { return s.confirmID != "" }

type routerModel struct {
	containerUC *usecase.ContainerUsecase
	imageUC     *usecase.ImageUsecase
	daemonUC    *usecase.DaemonUsecase

	width  int
	height int

	nav NavBar
	km  globalKeyMap

	currentPageID pageID
	currentPage   Page

	modal modalSession
}

// compile-time interface check
var _ tea.Model = routerModel{}

func New(containerUC *usecase.ContainerUsecase, imageUC *usecase.ImageUsecase, daemonUC *usecase.DaemonUsecase) tea.Model {
	gkm := newGlobalKeyMap()
	p := newOverviewPage(gkm, daemonUC)
	return routerModel{
		containerUC:   containerUC,
		imageUC:       imageUC,
		daemonUC:      daemonUC,
		nav:           NewNavBar(pageMetas()),
		km:            gkm,
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

	// グローバルキーの処理
	if handled, cmd := m.handleGlobalKeys(msg); handled {
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

func (m *routerModel) handleGlobalKeys(msg tea.Msg) (bool, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return false, nil
	}
	if key.Matches(km, m.km.Quit) {
		return true, tea.Quit
	}
	return false, nil
}

func (m *routerModel) handleDialog(msg tea.Msg) (bool, tea.Cmd) {
	// ダイアログのオープン
	switch x := msg.(type) {
	case openDialogMsg:
		m.modal = modalSession{
			dialog: newDialog(x.kind, x.title, x.body),
		}
		m.applyWindowSizeToDialog()
		return true, nil
	case openConfirmDialogMsg:
		m.modal = modalSession{
			dialog:    newDialog(dialogConfirm, x.title, x.body),
			confirmID: x.id,
		}
		m.applyWindowSizeToDialog()
		return true, nil
	}

	// モーダルが無ければ処理しない
	if m.modal.dialog == nil {
		return false, nil
	}

	// モーダル表示中はKeyMsgを他ページに流さない
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return true, nil
	}

	// 確認ダイアログの結果がokResultに格納される
	closed, okResult := m.modal.dialog.Update(km)
	if !closed {
		return true, nil
	}

	// モーダルのクローズ処理
	s := m.modal
	m.modal = modalSession{}
	// 確認モーダルでない場合はそのまま閉じる
	if !s.isConfirm() {
		return true, nil
	}
	// 確認モーダルの場合は結果を流す
	return true, func() tea.Msg {
		return confirmDialogResolvedMsg{id: s.confirmID, ok: okResult}
	}
}

func (m *routerModel) handleWindowSize(msg tea.Msg) (bool, tea.Cmd) {
	ws, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return false, nil
	}

	m.width, m.height = ws.Width, ws.Height
	m.applyWindowSizeToCurrentPage()
	m.applyWindowSizeToDialog()
	return true, nil
}

func (m routerModel) closeCurrentPageCmd() tea.Cmd {
	// Optional Close lifecycle hook
	if c, ok := m.currentPage.(PageCloser); ok {
		return c.Close()
	}
	return nil
}

func (m routerModel) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.km.NavOverview):
			return m, func() tea.Msg { return navigateMsg{to: pageOverview} }
		case key.Matches(msg, m.km.NavImages):
			return m, func() tea.Msg { return navigateMsg{to: pageImages} }
		case key.Matches(msg, m.km.NavContainers):
			return m, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}

	case openLogsMsg:
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageContainers
		m.currentPage = newLogsPage(m.km, m.containerUC, msg.id, msg.name)
		m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case navigateMsg:
		// Close the current page (if supported) right before navigation.
		// NOTE: closeCmd is generated before currentPage is replaced.
		closeCmd := m.closeCurrentPageCmd()
		switch msg.to {
		case pageOverview:
			m.currentPageID = pageOverview
			m.currentPage = newOverviewPage(m.km, m.daemonUC)
			m.applyWindowSizeToCurrentPage()
			return m, tea.Batch(closeCmd, m.currentPage.Init())

		case pageImages:
			m.currentPageID = pageImages
			m.currentPage = newImagesPage(m.km, m.imageUC)
			m.applyWindowSizeToCurrentPage()
			return m, tea.Batch(closeCmd, m.currentPage.Init())

		case pageContainers:
			m.currentPageID = pageContainers
			m.currentPage = newContainersPage(m.km, m.containerUC)
			m.applyWindowSizeToCurrentPage()
			return m, tea.Batch(closeCmd, m.currentPage.Init())
		}
	}

	updated, cmd := m.currentPage.Update(msg)
	m.currentPage = updated
	return m, cmd
}

func (m routerModel) dialogView() (string, bool) {
	if m.modal.dialog == nil {
		return "", false
	}
	return m.modal.dialog.View(), true
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
	if m.modal.dialog == nil {
		return
	}
	if m.width <= 0 || m.height <= 0 {
		return
	}
	_, _ = m.modal.dialog.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
}
