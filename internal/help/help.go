package help

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	_styles "github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/notes/internal/keymap"
)

var styles = _styles.New(_styles.IsDark())

type FullViewToggledMsg struct{}

type Model struct {
	Keys      keymap.Model
	Searching bool
	FullView  bool
	help      help.Model
	width     int
	height    int
	viewport  viewport.Model

	renderedFullView string
}

func (m *Model) getKeys() keymap.Model {
	if m.Searching {
		return keymap.SearchKeyMap
	}

	return m.Keys
}

func (m *Model) CombineWithDefaultKeys(keys keymap.Model) {
	m.Keys = keymap.CombineKeys(keymap.DefaultKeyMap, keys)
}

func (m *Model) SetKeyMap(keys keymap.Model) {
	m.Keys = keys
}

func New() Model {
	helpMenu := help.New()
	helpMenu.Styles = help.Styles{
		ShortKey:       styles.Info,
		ShortDesc:      styles.Overlay1,
		ShortSeparator: styles.Subtext0,
		FullKey:        styles.Subtext0,
		FullDesc:       styles.Overlay1,
		FullSeparator:  styles.Subtext0,
	}

	return Model{
		Keys: keymap.DefaultKeyMap,
		help: helpMenu,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) View() string {
	if m.FullView {
		return m.viewport.View() + m.getPercentageBar()
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(m.help.View(m.getKeys()))
}

func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.SetWidth(msg.Width)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Help):
			m.FullView = !m.FullView

			if m.FullView {
				m.renderedFullView = m.renderFullHelpView()
				height := min(lipgloss.Height(m.renderedFullView), m.height)
				m.viewport.SetHeight(height)
				m.viewport.SetWidth(m.width)
				m.viewport.SetContent(m.renderedFullView)
			}

			return *m, func() tea.Msg {
				return FullViewToggledMsg{}
			}
		}
	}

	return *m, cmd
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.help.SetWidth(width)

	if m.FullView {
		m.renderedFullView = m.renderFullHelpView()
		h := min(lipgloss.Height(m.renderedFullView), m.height)
		m.viewport.SetHeight(h)
		m.viewport.SetWidth(m.width)
		m.viewport.SetContent(m.renderedFullView)
	}
}

func (m *Model) renderFullHelpView() string {
	var sb strings.Builder

	bindings := keymap.ReplaceBinding(m.Keys.FullHelpBindings,
		key.NewBinding(
			key.WithKeys("f1", "alt+h"),
			key.WithHelp("f1 / alt+h", "close help"),
		),
	)

	enabledBindings := make([]key.Binding, 0)
	maxKeyWidth := 0

	bg := styles.Crust
	bgColour := bg.GetBackground()

	for _, binding := range bindings {
		if !binding.Enabled() {
			continue
		}

		enabledBindings = append(enabledBindings, binding)
		renderedWidth := lipgloss.Width(styles.Subtext1.Render(binding.Help().Key))
		maxKeyWidth = max(maxKeyWidth, renderedWidth)
	}

	for _, binding := range enabledBindings {
		keyText := binding.Help().Key
		renderedKey := styles.Info.Background(bgColour).Render(keyText)
		currentWidth := lipgloss.Width(renderedKey)
		padding := bg.Render(strings.Repeat(" ", maxKeyWidth-currentWidth+2))

		totalIndentation := 2 + lipgloss.Width(renderedKey) + maxKeyWidth - currentWidth + 2

		desc := strings.Split(binding.Help().Desc, "\n")

		var renderedDescription strings.Builder
		for i, line := range desc {
			renderedLine := styles.Overlay1.Background(bgColour).Render(strings.TrimSpace(line))

			if i != 0 {
				indentPadding := bg.Render(strings.Repeat(" ", totalIndentation))
				renderedDescription.WriteString("\n" + indentPadding)
			}

			renderedDescription.WriteString(renderedLine)
		}

		fmt.Fprintf(&sb, "• %s%s%s\n",
			renderedKey,
			padding,
			renderedDescription.String())
	}

	return bg.Width(m.width).Padding(1, 1).Render(strings.Trim(sb.String(), "\n"))
}

func (m *Model) getPercentageBar() string {
	scrollPercent := m.viewport.ScrollPercent()

	if lipgloss.Height(m.renderedFullView) <= m.height/2 {
		return ""
	}

	percentage := styles.Accent.Background(styles.Crust.GetBackground()).Render(fmt.Sprintf("%3.f%%", scrollPercent*100))

	return "+\n" + styles.Crust.Render(strings.Repeat(" ", m.width-lipgloss.Width(percentage))) + percentage
}
