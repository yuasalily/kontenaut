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
	// FooterKeys returns mode-specific footer bindings (excluding table navigation).
	//
	// Why exclude table navigation:
	// - Up/Down are always available and belong to the shared table component.
	// - Keeping them in the router prevents modes from depending on the mutable table model.
	FooterKeys(ctx imagesCtx) []key.Binding

	// Update interprets messages and returns:
	// - an action describing side effects the router should perform
	// - handled: whether this msg should be swallowed (i.e. not passed to table.Update)
	//
	// Why imagesView (read-only):
	// - Mode rules should not mutate shared state directly.
	// - Router remains the single owner of state and side effects.
	Update(ctx imagesCtx, v imagesView, msg tea.Msg) (act imagesAction, handled bool)
}

// imagesView is a read-only snapshot of UI-relevant state for mode logic.
//
// What:
// - Cursor and selection information needed to interpret keys.
// - "Selectable" for the cursor row in the current mode (locked/busy are filtered).
//
// Why:
// - Keeps mode submodels pure-ish: input -> action.
// - Avoids passing *imagesState (mutable maps/table/items) into models.
type imagesView struct {
	HasCursor        bool
	CursorID         string
	CursorSelectable bool

	SelectedIDs  []string
	HasSelection bool
}

type imagesAction interface{ isImagesAction() }

// actNone represents "do nothing".
type actNone struct{}

func (actNone) isImagesAction() {}

// actRefresh triggers list reload. Router decides whether selections is cleared.
type actRefresh struct{}

func (actRefresh) isImagesAction() {}

// actSwitchMode requests a mode transition.
//
// What:
// - Mode submodels request transitions by returning this action.
// Why:
// - Router owns transitions to keep selection clearing and table rebuild consistent.
// - Avoids "next mode" returns value and keeps Update() action-driven.
type actSwitchMode struct{ to imagesModeID }

func (actSwitchMode) isImagesAction() {}

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
