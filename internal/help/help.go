package help

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/styles"
)

type FullViewToggledMsg struct {
	Opened bool
}

func (m Model) getKeys() keymap.Model {
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

type Model struct {
	Keys      keymap.Model
	Searching bool
	FullView  bool

	help help.Model

	width  int
	height int
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

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	if m.FullView {
		return m.renderFullHelpView()
	}

	return lipgloss.NewStyle().Padding(0, 1).Render(m.help.View(m.getKeys()))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Help):
			m.FullView = !m.FullView

			return m, func() tea.Msg {
				return FullViewToggledMsg{
					Opened: m.FullView,
				}
			}
		}
	}

	return m, nil
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) renderFullHelpView() string {
	var sb strings.Builder

	bindings := keymap.ReplaceBinding(m.Keys.FullHelpBindings,
		key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "close help"),
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
		renderedDescription := styles.Overlay1.Background(bgColour).Render(binding.Help().Desc)
		currentWidth := lipgloss.Width(renderedKey)
		padding := bg.Render(strings.Repeat(" ", maxKeyWidth-currentWidth+2))

		sb.WriteString(fmt.Sprintf("• %s%s%s\n",
			renderedKey,
			padding,
			renderedDescription,
		))
	}

	return bg.Width(m.width).Padding(1, 1).Render(strings.Trim(sb.String(), "\n"))
}
