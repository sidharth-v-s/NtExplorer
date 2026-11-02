package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds every reusable Lip Gloss style. Colors adapt to light and dark
// terminals, and state is never communicated by color alone: selected rows
// also carry a marker and bold weight.
type Styles struct {
	Brand      lipgloss.Style
	HeaderMeta lipgloss.Style

	Panel        lipgloss.Style
	PanelFocused lipgloss.Style
	PanelTitle   lipgloss.Style
	PanelSub     lipgloss.Style

	Group    lipgloss.Style
	Row      lipgloss.Style
	RowDim   lipgloss.Style
	Selected lipgloss.Style
	Marker   lipgloss.Style

	Section      lipgloss.Style
	Label        lipgloss.Style
	Text         lipgloss.Style
	Accent       lipgloss.Style
	Code         lipgloss.Style
	Tag          lipgloss.Style
	Warning      lipgloss.Style
	Success      lipgloss.Style
	Error        lipgloss.Style
	Footer       lipgloss.Style
	FooterHi     lipgloss.Style
	Overlay      lipgloss.Style
	OverlayTitle lipgloss.Style
}

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#7aa2f7"}
	colorPurple = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#bb9af7"}
	colorGreen  = lipgloss.AdaptiveColor{Light: "#407010", Dark: "#9ece6a"}
	colorYellow = lipgloss.AdaptiveColor{Light: "#8a5a00", Dark: "#e0af68"}
	colorRed    = lipgloss.AdaptiveColor{Light: "#b3261e", Dark: "#f7768e"}
	colorText   = lipgloss.AdaptiveColor{Light: "#1f2335", Dark: "#c0caf5"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#5b6178", Dark: "#7f8bb3"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#c4c8da", Dark: "#3b4261"}
	colorSelBg  = lipgloss.AdaptiveColor{Light: "#dbe4ff", Dark: "#283457"}
)

// NewStyles builds the application style set.
func NewStyles() Styles {
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1)
	panelFocused := panel.BorderForeground(colorAccent)

	return Styles{
		Brand:      lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
		HeaderMeta: lipgloss.NewStyle().Foreground(colorMuted),

		Panel:        panel,
		PanelFocused: panelFocused,
		PanelTitle:   lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
		PanelSub:     lipgloss.NewStyle().Foreground(colorMuted),

		Group:    lipgloss.NewStyle().Foreground(colorMuted).Bold(true),
		Row:      lipgloss.NewStyle().Foreground(colorText),
		RowDim:   lipgloss.NewStyle().Foreground(colorMuted),
		Selected: lipgloss.NewStyle().Foreground(colorText).Background(colorSelBg).Bold(true),
		Marker:   lipgloss.NewStyle().Foreground(colorAccent).Bold(true),

		Section:  lipgloss.NewStyle().Bold(true).Foreground(colorPurple),
		Label:    lipgloss.NewStyle().Foreground(colorMuted),
		Text:     lipgloss.NewStyle().Foreground(colorText),
		Accent:   lipgloss.NewStyle().Foreground(colorAccent),
		Code:     lipgloss.NewStyle().Foreground(colorGreen),
		Tag:      lipgloss.NewStyle().Foreground(colorPurple),
		Warning:  lipgloss.NewStyle().Foreground(colorYellow),
		Success:  lipgloss.NewStyle().Foreground(colorGreen),
		Error:    lipgloss.NewStyle().Foreground(colorRed),
		Footer:   lipgloss.NewStyle().Foreground(colorMuted),
		FooterHi: lipgloss.NewStyle().Foreground(colorAccent),
		Overlay: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2),
		OverlayTitle: lipgloss.NewStyle().Bold(true).Foreground(colorAccent),
	}
}
