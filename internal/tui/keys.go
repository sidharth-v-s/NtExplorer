package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap centralizes every binding used by the application. Views never
// compare raw key strings, so rebinding happens here only.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Top      key.Binding
	Bottom   key.Binding
	HalfUp   key.Binding
	HalfDown key.Binding

	Search   key.Binding
	Escape   key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Enter    key.Binding

	Related    key.Binding
	Structures key.Binding
	Constants  key.Binding
	Favorite   key.Binding
	References key.Binding
	Export     key.Binding
	Copy       key.Binding
	Examples   key.Binding
	Help       key.Binding
	Quit       key.Binding

	Confirm key.Binding
	Cancel  key.Binding
}

// DefaultKeyMap returns the documented NtExplorer bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("PgDn", "page down"),
		),
		Top:      key.NewBinding(key.WithKeys("home"), key.WithHelp("Home", "top")),
		Bottom:   key.NewBinding(key.WithKeys("end"), key.WithHelp("End", "bottom")),
		HalfUp:   key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("Ctrl+U", "half page up")),
		HalfDown: key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("Ctrl+D", "half page down")),

		Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Escape:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "back")),
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "panels")),
		ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("Shift+Tab", "panels")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "open")),

		Related:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "related")),
		Structures: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "structures")),
		Constants:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "constants")),
		Favorite:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "favorite")),
		References: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "references")),
		Export:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Copy:       key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy")),
		Examples:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "examples")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),

		Confirm: key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),
		Cancel:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "cancel")),
	}
}

// Hints returns compact footer hints for the current width. Bindings that do
// not fit are dropped from the end.
func (k KeyMap) Hints() []string {
	return []string{
		"/ Search",
		"Tab Panels",
		"Enter Open",
		"y Copy",
		"g Related",
		"s Structures",
		"c Constants",
		"r References",
		"f Favorite",
		"e Export",
		"? Help",
		"q Quit",
	}
}
