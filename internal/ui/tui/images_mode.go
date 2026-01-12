package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

// imagesMode is a submodel for the Images page.
//
// What:
// - Normal / Delete mode are separate submodules.
// - Each mode owns its own key interpretation and UI shape (columns/rows/footer).
//
// Why:
// - The spec is mode-centric (SEL column only in mode, enter=execute in mode, lock rules).
// - Keeping mode rules localized prevents branching from spreading in the router.
type imagesMode interface {
	ID() imagesModeID
	Title() string
	Columns(totalWidth int) []table.Column
	Rows(st *imagesState) []table.Row
	FooterKeys(ctx imagesCtx, st *imagesState) []key.Binding

	// Update interprets messages and returns:
	// - next mode (optional)
	// - an action describing side effects the router should perform
	// - handled: whether this msg should be swallowed (i.e. not passed to table.Update)
	Update(ctx imagesCtx, st *imagesState, msg tea.Msg) (next imagesMode, act imagesAction, handled bool)
}

type imagesAction interface{ isImagesAction() }

// actNone represents "do nothing".
type actNone struct{}

func (actNone) isImagesAction() {}

// actRefresh triggers list reload. Router decides whether selections is cleared.
type actRefresh struct{}

func (actRefresh) isImagesAction() {}

// actEnterDeleteMode switches mode to delete.
type actEnterDeleteMode struct{}

func (actEnterDeleteMode) isImagesAction() {}

// actExitMode exits the current mode and returns to normal.
type actExitMode struct{}

func (actExitMode) isImagesAction() {}

// actRequestDelete asks the router to open a confirm dialog for deleting ids.
//
// Note:
// - Normal 'd' and Delete-mode 'enter' both end up here.
// - Router re-validates lock/busy at execution time to handle racey state changes.
type actRequestDelete struct{ ids []string }

func (actRequestDelete) isImagesAction() {}

// actToggleSelect toggles selection for the given id.
type actToggleSelect struct{ id string }

func (actToggleSelect) isImagesAction() {}

// actExecuteDelete means "execute delete for the current selection" (delete mode only).
type actExecuteDelete struct{}

func (actExecuteDelete) isImagesAction() {}
