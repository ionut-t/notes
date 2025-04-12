package ui

import (
	"math"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

type addView int

const (
	addContent addView = iota
	addName
	abbortAdd
)

type updateValueMsg []byte

type AddModel struct {
	store         *note.Store
	width, height int
	err           error
	success       bool
	view          addView
	content       *huh.Text
	filename      *huh.Input
	filenameError error
	help          help.Model
	standalone    bool
	active        bool
}

func NewAddModel(store *note.Store) AddModel {
	content := huh.NewText().
		Key("content").
		Placeholder("Write your note here").
		ShowLineNumbers(true).
		CharLimit(math.MaxInt64)

	fileName := huh.NewInput().
		Key("fileName").
		Title("Note name").
		Placeholder("Enter a name for your note").
		Validate(huh.ValidateLength(1, 20))

	fileName.WithTheme(styles.ThemeCatppuccin())
	content.WithTheme(styles.ThemeCatppuccin())
	content.WithWidth(80)
	content.WithHeight(10)
	content.Focus()

	content.WithKeyMap(&huh.KeyMap{
		Text: huh.TextKeyMap{
			NewLine: keymap.NewLine,
		},
		Quit: keymap.QuitForm,
	})

	helpMenu := help.New()
	helpMenu.SetKeyMap(keymap.DefaultKeyMap)

	m := AddModel{
		store:      store,
		view:       addContent,
		content:    content,
		filename:   fileName,
		help:       helpMenu,
		standalone: true,
		active:     true,
	}

	m.setHelp()

	return m
}

func (m *AddModel) markAsIntegrated() {
	m.standalone = false
	m.content.WithHeight(max(m.height-4, 10))
}

func (m AddModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tea.SetWindowTitle("Notes"))
}

func (m AddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.content.WithWidth(m.width - 4)
		m.content.WithHeight(min(m.height-4, 10))

	case updateValueMsg:
		value := string(msg)
		m.content.Value(&value)

		content, cmd := m.content.Update(msg)
		m.content = content.(*huh.Text)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.ForceQuit):
			m.view = abbortAdd
			return m, tea.Quit

		case key.Matches(msg, keymap.Cancel):
			if m.view == addName {
				m.view = addContent
				m.setHelp()
				m.content.Focus()

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

		case key.Matches(msg, keymap.Editor):
			if m.view == addName {
				break
			}

			tmpFile, _ := os.CreateTemp(os.TempDir(), "*.md")
			tmpFile.WriteString(m.content.GetValue().(string))

			execCmd := tea.ExecProcess(exec.Command(m.store.GetEditor(), tmpFile.Name()), func(error) tea.Msg {
				content, _ := os.ReadFile(tmpFile.Name())
				_ = os.Remove(tmpFile.Name())
				return updateValueMsg(content)
			})

			return m, execCmd

		case key.Matches(msg, keymap.Save):
			if m.view == addName {
				content := m.content.GetValue().(string)

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
		}
	}

	switch m.view {
	case addContent:
		content, cmd := m.content.Update(msg)
		m.content = content.(*huh.Text)
		cmds = append(cmds, cmd)

	case addName:
		fileName, cmd := m.filename.Update(msg)
		m.filename = fileName.(*huh.Input)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AddModel) View() string {
	if m.view == abbortAdd {
		return ""
	}

	return lipgloss.NewStyle().Width(m.width).Padding(1, 2).Render(m.getView())
}

func (m AddModel) getView() string {
	if m.err != nil {
		return styles.Error.Render(m.err.Error())
	}

	if m.success {
		return styles.Success.Render("Note created successfully!")
	}

	switch m.view {
	case addContent:
		return m.content.View() + "\n\n" + m.help.View()
	case addName:
		if err := m.filenameError; err != nil {
			return m.filename.View() + "\n" + styles.Error.Render(err.Error()) + "\n\n" + m.help.View()
		}

		return m.filename.View() + "\n\n" + m.help.View()
	default:
		return ""
	}
}

func (m *AddModel) setHelp() {
	switch m.view {
	case addContent:
		if m.standalone {
			m.help.Keys.ShortHelpBindings = []key.Binding{
				keymap.NewLine,
				keymap.Editor,
				keymap.Continue,
				keymap.Back,
			}
		} else {
			m.help.Keys.ShortHelpBindings = []key.Binding{
				keymap.NewLine,
				keymap.Editor,
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

	content := m.content.GetValue().(string)
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
