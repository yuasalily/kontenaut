package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yuasalily/kontenaut/internal/usecase"
)

// modalSession tracks the current modal dialog state (if any)
type modalSession struct {
	dialog *dialogModel
	yesMsg tea.Msg
	noMsg  tea.Msg
}

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

	// globalBusy blocks router-level navigation (1/2/3) during destructive operations.
	// Quit and dialogs remain available.
	globalBusy bool
}

// compile-time interface check
var _ tea.Model = routerModel{}

// New constructs the root Bubble Tea model(router).
//
// Why:
// - The router owns global concerns (navigation, window size, modal dialogs).
// - Each page remains focused on its own UI/state and domain usecases.
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
	// Why ordering:
	// Window size affects layout everywhere (including modal), so handle it first.
	// - Global keys (quit/navigation) should work consistently across pages.
	// - Dialogs should intercept key events and prevent them from reaching pages.

	// ウィンドウサイズの処理
	if nm, handled, cmd := m.handleWindowSize(msg); handled {
		return nm, cmd
	}

	// グローバルキーの処理
	if nm, handled, cmd := m.handleGlobalKeys(msg); handled {
		return nm, cmd
	}

	// ダイアログの処理
	if nm, handled, cmd := m.handleDialog(msg); handled {
		return nm, cmd
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

func (m routerModel) handleWindowSize(msg tea.Msg) (routerModel, bool, tea.Cmd) {
	ws, ok := msg.(tea.WindowSizeMsg)
	if !ok {
		return m, false, nil
	}

	m.width, m.height = ws.Width, ws.Height
	m = m.applyWindowSizeToCurrentPage()
	m = m.applyWindowSizeToDialog()
	return m, true, nil
}

func (m routerModel) handleGlobalKeys(msg tea.Msg) (routerModel, bool, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, false, nil
	}
	if key.Matches(km, m.km.Quit) {
		return m, true, tea.Quit
	}
	return m, false, nil
}

func (m routerModel) handleDialog(msg tea.Msg) (routerModel, bool, tea.Cmd) {
	// ダイアログのオープン
	switch x := msg.(type) {
	case openDialogMsg:
		m.modal = modalSession{
			dialog: newDialog(x.kind, x.title, x.body),
		}
		m = m.applyWindowSizeToDialog()
		return m, true, nil
	case openConfirmDialogMsg:
		m.modal = modalSession{
			dialog: newDialog(dialogConfirm, x.title, x.body),
			yesMsg: x.yesMsg,
			noMsg:  x.noMsg,
		}
		m = m.applyWindowSizeToDialog()
		return m, true, nil
	}

	// モーダルが無ければ処理しない
	if m.modal.dialog == nil {
		return m, false, nil
	}

	// モーダル表示中はKeyMsgを他ページに流さない
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, true, nil
	}

	// 確認ダイアログの結果がokResultに格納される
	closed, okResult := m.modal.dialog.Update(km)
	if !closed {
		return m, true, nil
	}

	// モーダルのクローズ処理
	s := m.modal
	m.modal = modalSession{}

	if okResult {
		if s.yesMsg == nil {
			return m, true, nil
		}
		return m, true, func() tea.Msg { return s.yesMsg }
	}
	if s.noMsg == nil {
		return m, true, nil
	}
	return m, true, func() tea.Msg { return s.noMsg }
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
	case setGlobalBusyMsg:
		m.globalBusy = msg.on
		return m, nil

	case tea.KeyMsg:
		// Block router-level navigation while busy.
		// Note: quit is handled earlier by handleGlobalKeys and remains available.
		switch {
		case key.Matches(msg, m.km.NavOverview) && !m.globalBusy:
			return m, func() tea.Msg { return navigateMsg{to: pageOverview} }
		case key.Matches(msg, m.km.NavImages) && !m.globalBusy:
			return m, func() tea.Msg { return navigateMsg{to: pageImages} }
		case key.Matches(msg, m.km.NavContainers) && !m.globalBusy:
			return m, func() tea.Msg { return navigateMsg{to: pageContainers} }
		}

	case openImagesDeleteMsg:
		// Keep currentPageID as pageImages so the navbar stays on "Images".
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageImages
		m.currentPage = newImagesDeletePage(m.km, m.imageUC)
		m = m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case openContainersDeleteMsg:
		// Keep currentPageID as pageContainers so the navbar stays on "Containers".
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageContainers
		m.currentPage = newContainersDeletePage(m.km, m.containerUC)
		m = m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case openContainersStartMsg:
		// Keep currentPageID as pageContainers so the navbar stays on "Containers".
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageContainers
		m.currentPage = newContainersStartPage(m.km, m.containerUC)
		m = m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case openContainersStopMsg:
		// Keep currentPageID as pageContainers so the navbar stays on "Containers".
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageContainers
		m.currentPage = newContainersStopPage(m.km, m.containerUC)
		m = m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case openLogsMsg:
		// Why logs are opened via Msg:
		// - Keeps navigation page-agnostic (pages emit messages; router decides transitions).
		// - Allows including immutable data (container ID/name) at the time of selection.
		closeCmd := m.closeCurrentPageCmd()
		m.currentPageID = pageContainers
		m.currentPage = newLogsPage(m.km, m.containerUC, msg.id, msg.name)
		m = m.applyWindowSizeToCurrentPage()
		return m, tea.Batch(closeCmd, m.currentPage.Init())

	case navigateMsg:
		// Close the current page (if supported) right before navigation.
		// Note: closeCmd is generated before currentPage is replaced.
		//
		// Why:
		// - Some pages start background tasks (e.g. log streaming).
		// - Close must be created before overwriting currentPage.
		closeCmd := m.closeCurrentPageCmd()
		switch msg.to {
		case pageOverview:
			m.currentPageID = pageOverview
			m.currentPage = newOverviewPage(m.km, m.daemonUC)
			m = m.applyWindowSizeToCurrentPage()
			return m, tea.Batch(closeCmd, m.currentPage.Init())

		case pageImages:
			m.currentPageID = pageImages
			m.currentPage = newImagesPage(m.km, m.imageUC)
			m = m.applyWindowSizeToCurrentPage()
			return m, tea.Batch(closeCmd, m.currentPage.Init())

		case pageContainers:
			m.currentPageID = pageContainers
			m.currentPage = newContainersPage(m.km, m.containerUC)
			m = m.applyWindowSizeToCurrentPage()
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

func (m routerModel) applyWindowSizeToCurrentPage() routerModel {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	h := max(m.height-m.nav.Height(), 1)
	updated, _ := m.currentPage.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: h,
	})
	m.currentPage = updated
	return m
}

func (m routerModel) applyWindowSizeToDialog() routerModel {
	if m.modal.dialog == nil {
		return m
	}
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	_, _ = m.modal.dialog.Update(tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	})
	return m
}
