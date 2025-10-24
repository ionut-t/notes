package ui

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type addView int

const (
	addContent addView = iota
	addName
	abbortAdd
)

var addViewBorder = lipgloss.Border{
	Left: "│",
}

type updateValueMsg []byte

type AddModel struct {
	store            *note.Store
	width, height    int
	err              error
	success          bool
	view             addView
	editor           editor.Model
	filename         *huh.Input
	confirmation     *huh.Confirm
	filenameError    error
	help             help.Model
	standalone       bool
	active           bool
	showConfirmation bool
}

func NewAddModel(store *note.Store) AddModel {
	textEditor := editor.New(80, 10)
	textEditor.SetCursorMode(editor.CursorBlink)
	textEditor.SetInsertMode()
	textEditor.DisableCommandMode(true)
	textEditor.SetLanguage("markdown", styles.EditorLanguageTheme())
	textEditor.SetExtraHighlightedContextLines(1000)
	textEditor.Focus()

	fileName := huh.NewInput().
		Key("fileName").
		Title("Note name").
		Placeholder("Enter a name for your note").
		Validate(huh.ValidateLength(1, 20))

	confirmation := huh.NewConfirm().
		Title("You have unsaved changes. Are you sure you want to quit?").
		Affirmative("Yes").
		Negative("No")

	fileName.WithTheme(styles.ThemeCatppuccin())

	confirmation.WithKeyMap(&huh.KeyMap{
		Confirm: huh.NewDefaultKeyMap().Confirm,
	})

	confirmation.WithTheme(styles.ThemeCatppuccin())

	helpMenu := help.New()
	helpMenu.SetKeyMap(keymap.DefaultKeyMap)

	m := AddModel{
		store:        store,
		view:         addContent,
		editor:       textEditor,
		filename:     fileName,
		confirmation: confirmation,
		help:         helpMenu,
		standalone:   true,
		active:       true,
	}

	m.setHelp()

	return m
}

func (m *AddModel) markAsIntegrated() {
	m.standalone = false
	m.setContentHeight()
	em, _ := m.editor.Update(nil)
	m.editor = em.(editor.Model)
	m.editor.SetNormalMode()
	m.filename.WithWidth(min(m.width-2, 50))
}

func (m AddModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.editor.CursorBlink(), tea.SetWindowTitle("Notes"))
}

func (m AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.editor.SetSize(m.width-4, min(m.height-4, 20))
		m.filename.WithWidth(min(m.width-4, 50))

	case updateValueMsg:
		m.editor.SetBytes(msg)

		content, cmd := m.editor.Update(msg)
		m.editor = content.(editor.Model)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.ForceQuit):
			m.view = abbortAdd
			return m, tea.Quit

		case key.Matches(msg, keymap.Cancel):
			if m.view == addContent && !m.editor.IsNormalMode() {
				break
			}

			if m.showConfirmation {
				m.showConfirmation = false
				m.editor.Focus()
				break
			}

			if m.hasChanges() && m.view == addContent {
				m.showConfirmation = true
				m.editor.Blur()
				m.confirmation.Focus()
				break
			}

			if m.view == addName {
				m.view = addContent
				m.setHelp()
				m.editor.Focus()

				if _, err := validateNoteName(m.filename); err == nil {
					m.filenameError = nil
					break
				}
			} else {
				m.active = false

				if m.standalone {
					m.view = abbortAdd
					return m, tea.Quit
				}

				return m, dispatch(cmdAbortMsg{})
			}

		case key.Matches(msg, keymap.Continue):
			if m.view == addContent {
				m.view = addName
				m.setHelp()
				m.setName()

				m.filename.Focus()
			}

		case key.Matches(msg, keymap.ExternalEditor):
			if m.view == addName {
				break
			}

			tmpFile, _ := os.CreateTemp(os.TempDir(), "*.md")
			tmpFile.WriteString(m.editor.GetCurrentContent())

			execCmd := tea.ExecProcess(exec.Command(m.store.GetEditor(), tmpFile.Name()), func(error) tea.Msg {
				content, _ := os.ReadFile(tmpFile.Name())
				_ = os.Remove(tmpFile.Name())
				return updateValueMsg(content)
			})

			return m, execCmd

		case key.Matches(msg, keymap.Save):
			if m.showConfirmation {
				confirmed := m.confirmation.GetValue().(bool)

				if confirmed {
					m.active = false
					m.showConfirmation = false

					if m.standalone {
						m.view = abbortAdd
						return m, tea.Quit
					}

					return m, dispatch(cmdAbortMsg{})
				}

				m.showConfirmation = false

				m.editor.Focus()

				break
			}

			if m.view == addName {
				content := m.editor.GetCurrentContent()

				noteName, err := validateNoteName(m.filename)

				if err != nil {
					m.filenameError = err
					break
				}

				m.filenameError = nil

				noteName = strings.Join(strings.Split(noteName, " "), "-")

				if err := m.store.Create(noteName, content); err != nil {
					m.err = err

					if !m.standalone {
						return m, dispatch(cmdErrorMsg(err))
					}
				} else {
					m.active = false

					if m.standalone {
						m.success = true

						return m, tea.Quit
					}

					return m, dispatch(noteAddedMsg{})
				}
			}

		case key.Matches(msg, keymap.Accept):
			if m.showConfirmation {
				m.showConfirmation = false

				m.active = false
				m.showConfirmation = false

				if m.standalone {
					m.view = abbortAdd
					return m, tea.Quit
				}

				return m, dispatch(cmdAbortMsg{})
			}

		case key.Matches(msg, keymap.Reject):
			if m.showConfirmation {
				m.showConfirmation = false
				m.editor.Focus()

				return m, nil
			}
		}
	}

	switch m.view {
	case addContent:
		content, cmd := m.editor.Update(msg)
		m.editor = content.(editor.Model)
		cmds = append(cmds, cmd)

	case addName:
		fileName, cmd := m.filename.Update(msg)
		m.filename = fileName.(*huh.Input)
		cmds = append(cmds, cmd)
	}

	if m.showConfirmation {
		confirmation, cmd := m.confirmation.Update(msg)
		m.confirmation = confirmation.(*huh.Confirm)
		cmds = append(cmds, cmd)
	}

	if !m.standalone {
		m.setContentHeight()
	}

	return m, tea.Batch(cmds...)
}

func (m AddModel) View() string {
	if m.view == abbortAdd {
		return ""
	}

	if m.standalone {
		return lipgloss.NewStyle().
			Border(addViewBorder).
			BorderForeground(styles.Accent.GetForeground()).
			Padding(1, 2).
			Margin(0, 1).
			Render(m.getView())
	}

	return lipgloss.NewStyle().
		Padding(1, 1).
		Render(m.getView())
}

func (m AddModel) getView() string {
	if m.err != nil {
		return styles.Error.Render(m.err.Error())
	}

	if m.success {
		return styles.Success.Render("Note created successfully!")
	}

	footer := utils.Ternary(m.showConfirmation, m.confirmation.View(), m.help.View())

	switch m.view {
	case addContent:
		return m.editor.View() + "\n\n" + footer
	case addName:
		if err := m.filenameError; err != nil {
			return m.filename.View() + "\n" + styles.Error.Render(err.Error()) + "\n\n" + footer
		}

		return m.filename.View() + "\n\n" + footer
	default:
		return ""
	}
}

func (m *AddModel) setHelp() {
	switch m.view {
	case addContent:
		if m.standalone {
			m.help.Keys.ShortHelpBindings = []key.Binding{
				keymap.ExternalEditor,
				keymap.Continue,
				keymap.Back,
			}
		} else {
			m.help.Keys.ShortHelpBindings = []key.Binding{
				keymap.ExternalEditor,
				keymap.Continue,
				keymap.Quit,
			}
		}

	case addName:
		m.help.Keys.ShortHelpBindings = []key.Binding{
			keymap.Save,
			keymap.Back,
			keymap.ForceQuit,
		}
	}
}

func (m *AddModel) setName() {
	currentName := m.filename.GetValue().(string)

	if currentName != "" {
		return
	}

	content := m.editor.GetCurrentContent()
	name := strings.Split(content, "\n")[0]

	if strings.HasPrefix(name, "#") {
		name = strings.Trim(name, "#")
		name = strings.TrimSpace(name)
		// Replace any sequence of dashes, spaces, or # with a single space
		re := regexp.MustCompile(`[#\s-]+`)
		name = re.ReplaceAllString(name, " ")
		name = strings.Join(strings.Fields(name), "-")
		name = strings.ToLower(name)

		m.filename.Value(&name)
	}
}

func (m *AddModel) hasChanges() bool {
	return len(m.editor.GetCurrentContent()) > 0
}

func (m *AddModel) blink() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m *AddModel) setContentHeight() {
	height := utils.Ternary(m.showConfirmation,
		m.height-5-lipgloss.Height(m.confirmation.View()),
		m.height-4,
	)

	m.editor.SetSize(m.width, max(height, 10))
}
