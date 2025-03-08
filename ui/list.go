package ui

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/internal/utils"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type listView int

const (
	listViewMain listView = iota
	noteView
	abortView
)

type ListModel struct {
	store                  *note.NotesStore
	list                   list.Model
	notes                  []note.Note
	view                   listView
	selectedNote           note.Note
	noteView               NoteModel
	error                  error
	showDeleteConfirmation bool
	renameInput            *huh.Input
	renameInputError       error
	renameMode             bool
	help                   help.Model
	width, height          int
	successMessage         string
	cmdInput               *huh.Input
	cmdMode                bool
}

func NewListModel(store *note.NotesStore) *ListModel {
	notes, err := store.GetAllNotes()

	if err != nil {
		fmt.Println("Error getting notes:", err)
		os.Exit(1)
	}

	items := processNotes(notes)

	delegate := list.NewDefaultDelegate()

	delegate.Styles = styles.ListItemStyles()

	m := ListModel{
		store:    store,
		list:     list.New(items, delegate, 0, 0),
		notes:    notes,
		help:     help.New(),
		cmdInput: huh.NewInput().Prompt(": "),
	}
	m.list.Title = "Notes"

	m.list.Styles = styles.ListStyles()

	m.list.FilterInput.PromptStyle = styles.Accent
	m.list.FilterInput.Cursor.Style = styles.Accent

	m.list.InfiniteScrolling = true
	m.list.SetShowHelp(false)

	m.help.Keys.FullHelpBindings = []key.Binding{
		keymap.Up,
		keymap.Down,
		keymap.Select,
		keymap.QuickEditor,
		keymap.Rename,
		keymap.Search,
		keymap.Copy,
		keymap.Delete,
		keymap.Quit,
		keymap.Help,
	}

	return &m
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func (m ListModel) Init() tea.Cmd {
	return tea.SetWindowTitle("Notes")
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case help.FullViewToggledMsg:
		return m, m.dispatchWindowSizeMsg()

	case editorFinishedMsg:
		return m.handleEditorClose()

	case deleteNoteMsg:
		return m.executeNoteDeletion(msg)

	case clearMsg:
		m.successMessage = ""
		m.noteView.successMessage = ""
		m.error = nil
		m.noteView.error = nil

	case tea.KeyMsg:
		keyMsg := tea.KeyMsg(msg).String()

		if m.renameMode {
			return m.handleRenameNote(tea.KeyMsg(msg))
		}

		if m.cmdMode {
			return m.handleCmdRunner(tea.KeyMsg(msg))
		}

		if m.list.FilterState() == list.Filtering {
			break
		}

		if m.showDeleteConfirmation {
			switch keyMsg {
			case "y", "Y":
				return m, dispatchDeleteMsg(true)
			case "n", "N", "esc", "q":
				return m, dispatchDeleteMsg(false)
			default:
				return m, nil
			}
		}

		switch keyMsg {
		case "esc", "q":
			return m.handleQuit()

		case "ctrl+d":
			return m.handleDeleteNote()

		case "enter":
			m.handleSelection()

		case "e":
			if ok, cmd := m.triggerNoteEditor(); ok {
				return m, cmd
			}

		case "r":
			return m.activateRenameMode()

		case "c":
			return m.copyNoteContent()

		case ":":
			m.cmdMode = true
			m.noteView.setHeight(m.height - lipgloss.Height(m.cmdInput.View()))
		}
	}

	switch m.view {
	case listViewMain:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)

		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel.(help.Model)
		cmds = append(cmds, cmd)

	case noteView:
		noteViewModel, cmd := m.noteView.Update(msg)
		m.noteView = noteViewModel.(NoteModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ListModel) View() string {
	switch m.view {
	case listViewMain:
		if m.showDeleteConfirmation {
			return docStyle.Render(m.list.View()) + "\n" + m.deleteConfirmationView()
		}

		if m.renameMode {
			return m.getViewInRenameMode(docStyle.Render(m.list.View()))
		}

		if m.cmdMode {
			return m.getViewInCmdMode()
		}

		return docStyle.Render(m.list.View()) + "\n" + m.statusBarView()

	case noteView:
		if m.renameMode {
			return m.getViewInRenameMode(m.noteView.View())
		}

		if m.cmdMode {
			return m.noteView.View() + "\n" + m.cmdInput.View()
		}

		return m.noteView.View()

	default:
		return ""
	}
}

func (m ListModel) getViewInRenameMode(mainView string) string {
	if m.renameInputError == nil {
		return mainView + "\n" + m.renameInput.View()
	}

	return mainView + "\n" + m.renameInput.View() + "\n" + styles.Error.Margin(0, 2).Render(m.renameInputError.Error())
}

func (m ListModel) getViewInCmdMode() string {
	mainView := docStyle.Render(m.list.View())

	if m.error == nil {
		return mainView + "\n" + m.cmdInput.View()
	}

	return mainView + "\n" + m.cmdInput.View() + "\n" + styles.Error.Margin(0, 2).Render(m.error.Error())
}

func (m ListModel) statusBarView() string {
	if m.successMessage != "" {
		return styles.Success.Padding(0, 1).Render(m.successMessage)
	}

	if m.list.FilterState() == list.Filtering {
		m.help.Keys.ShortHelpBindings = []key.Binding{
			keymap.Cancel,
		}
	} else {
		m.help.Keys.ShortHelpBindings = []key.Binding{
			keymap.Select,
			keymap.QuickEditor,
			keymap.Rename,
			keymap.Search,
			keymap.Delete,
			keymap.Quit,
			keymap.Help,
		}
	}

	if m.showDeleteConfirmation {
		return ""
	}

	if m.help.FullView {
		return m.help.View()
	}

	return lipgloss.NewStyle().Margin(0, 2).Render(m.help.View())
}

func processNotes(notes []note.Note) []list.Item {
	items := make([]list.Item, len(notes))

	slices.SortStableFunc(notes, func(i, j note.Note) int {
		if i.UpdatedAt.After(j.UpdatedAt) {
			return -1
		}

		if i.UpdatedAt.Before(j.UpdatedAt) {
			return 1
		}

		return 0
	})

	for i, n := range notes {
		items[i] = item{
			title: n.Name,
			desc:  fmt.Sprintf("Last modified: %s", n.UpdatedAt.Format("02/01/2006 15:04")),
		}
	}

	return items
}

func (m ListModel) deleteConfirmationView() string {
	bg := styles.Crust.GetBackground()

	question := styles.Text.
		Background(bg).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(1, 2).
		Render(fmt.Sprintf(
			"Are you sure you want to delete %s",
			styles.Accent.Background(bg).Render(m.list.Items()[m.list.Index()].(item).title)) + styles.Crust.Render("?"),
		)

	options := styles.Text.
		Background(bg).
		Width(m.width).
		Align(lipgloss.Center).
		Padding(0, 2, 1).
		Render(
			styles.Error.Background(bg).Render("[Y]es") + styles.Crust.Render(" | ") + styles.Crust.Render("[N]o"),
		)

	content := lipgloss.JoinVertical(lipgloss.Center, question, options)

	container := styles.Crust.
		Width(m.width).
		Align(lipgloss.Center).
		Render(content)

	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, container)
}

func dispatchDeleteMsg(confirmed bool) tea.Cmd {
	return func() tea.Msg {
		return deleteNoteMsg{
			confirmed: confirmed,
		}
	}
}

func (m *ListModel) handleWindowSize(msg tea.WindowSizeMsg) {
	h, v := docStyle.GetFrameSize()
	m.list.SetSize(msg.Width-h, msg.Height-v-lipgloss.Height(m.statusBarView()))
	m.width, m.height = msg.Width, msg.Height
	m.help.SetSize(msg.Width, msg.Height)

	if m.view == noteView {
		m.noteView.setHeight(msg.Height)
	}
}

func (m ListModel) handleEditorClose() (ListModel, tea.Cmd) {
	notes, err := m.store.GetAllNotes()

	if err != nil {
		fmt.Println("Error getting notes:", err)
		os.Exit(1)
	}

	m.list.SetItems(processNotes(notes))
	m.notes = notes
	m.selectedNote = notes[m.list.Index()]

	if m.view == noteView {
		m.noteView.setNote(m.selectedNote)
	}

	return m, func() tea.Msg {
		return tea.EnableMouseCellMotion()
	}
}

func (m ListModel) handleDeleteNote() (ListModel, tea.Cmd) {
	if m.view == noteView || len(m.list.Items()) == 0 {
		return m, nil
	}

	m.showDeleteConfirmation = true

	_, v := docStyle.GetFrameSize()
	m.list.SetHeight(m.height - v - lipgloss.Height(m.deleteConfirmationView()))

	return m, nil
}

func (m ListModel) executeNoteDeletion(msg deleteNoteMsg) (ListModel, tea.Cmd) {
	if msg.confirmed {
		selected := m.list.Index()

		noteName := m.list.Items()[selected].(item).title
		err := m.store.Delete(noteName)

		if err != nil {
			fmt.Println("Error deleting note:", err)
			os.Exit(1)
		}

		m.list.RemoveItem(selected)

		selected = m.list.Index()
		notes := slices.DeleteFunc(m.notes, func(n note.Note) bool {
			return n.Name == noteName
		})
		m.notes = notes
		m.selectedNote = m.notes[selected]
		m.successMessage = "Note successfully deleted"
	}

	m.showDeleteConfirmation = false

	return m, tea.Batch(
		m.dispatchWindowSizeMsg(),
		dispatchClearMsg(),
	)
}

func (m ListModel) handleRenameNote(msg tea.KeyMsg) (ListModel, tea.Cmd) {
	var cmds []tea.Cmd
	keyMsg := tea.KeyMsg(msg).String()

	inputModel, cmd := m.renameInput.Update(msg)
	m.renameInput = inputModel.(*huh.Input)
	cmds = append(cmds, cmd)

	switch keyMsg {
	case "enter":
		selected := m.list.Index()
		note := m.notes[selected]

		newName, err := validateNoteName(m.renameInput)

		if err != nil {
			m.renameInputError = err
			return m, nil
		}

		fileName, err := m.store.RenameNote(note.Name, newName)

		if err != nil {
			fmt.Println("Error renaming note:", err)
			os.Exit(1)
		}

		m.list.SetItem(selected, item{
			title: fileName,
			desc:  fmt.Sprintf("Last modified: %s", note.UpdatedAt.Format("02/01/2006 15:04")),
		})

		m.notes[selected].Name = fileName
		m.selectedNote = m.notes[selected]

		m.renameMode = false

		if m.view == noteView {
			m.noteView.setNote(m.selectedNote)
		}

		m.successMessage = "Note successfully renamed"
		m.noteView.successMessage = m.successMessage

		return m, tea.Batch(
			m.dispatchWindowSizeMsg(),
			dispatchClearMsg(),
		)

	case "esc":
		m.renameMode = false
		m.list.SetHeight(m.list.Height() + lipgloss.Height(m.renameInput.View()))

		return m, m.dispatchWindowSizeMsg()
	}

	return m, tea.Batch(cmds...)
}

func (m ListModel) handleQuit() (ListModel, tea.Cmd) {
	if m.renameMode {
		return m, nil
	}

	if m.view == noteView {
		m.view = listViewMain
		return m, nil
	}

	m.view = abortView
	return m, tea.Quit
}

func (m *ListModel) handleSelection() {
	if m.view != listViewMain || len(m.list.Items()) == 0 {
		return
	}

	selected := m.list.Index()
	note := m.notes[selected]

	m.selectedNote = note
	m.noteView = NewNoteModel(note, m.width, m.height)

	m.view = noteView
}

func (m *ListModel) triggerNoteEditor() (bool, tea.Cmd) {
	if m.showDeleteConfirmation || len(m.list.Items()) == 0 {
		return false, nil
	}

	selected := m.list.Index()
	note := m.notes[selected]
	m.selectedNote = note

	notePath := m.store.GetNotePath(m.selectedNote.Name)
	execCmd := tea.ExecProcess(exec.Command(m.store.GetEditor(), notePath), func(error) tea.Msg {
		return editorFinishedMsg{}
	})

	return true, execCmd
}

func (m ListModel) activateRenameMode() (ListModel, tea.Cmd) {
	m.renameMode = true
	m.help.FullView = false
	m.noteView.help.FullView = false
	m.renameInputError = nil

	selected := m.list.Index()
	note := m.notes[selected]
	m.renameInput = huh.NewInput().Title("Rename " + note.Name).Prompt("New name: ")
	m.renameInput.WithTheme(styles.ThemeCatppuccin())
	m.renameInput.Focus()

	if m.view == listViewMain {
		_, v := docStyle.GetFrameSize()
		m.list.SetHeight(m.height - v - lipgloss.Height(m.renameInput.View()))
	} else {
		m.noteView.setHeight(m.height - lipgloss.Height(m.renameInput.View()))
	}

	return m, nil
}

func (m ListModel) copyNoteContent() (ListModel, tea.Cmd) {
	err := m.store.CopyContent(m.selectedNote.Content)
	if err != nil {
		fmt.Println("Error copying note:", err)
		os.Exit(1)
	}

	m.successMessage = "Note copied to clipboard"
	m.noteView.successMessage = m.successMessage

	return m, dispatchClearMsg()
}

func dispatchClearMsg() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return clearMsg{}
	})
}

func (m ListModel) dispatchWindowSizeMsg() tea.Cmd {
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: m.width, Height: m.height}
	}
}

func (m ListModel) handleCmdRunner(msg tea.KeyMsg) (ListModel, tea.Cmd) {
	m.cmdInput.Focus()

	keyMsg := tea.KeyMsg(msg).String()
	switch keyMsg {
	case "esc":
		m.cmdMode = false
		empty := ""
		m.error = nil
		m.cmdInput.Value(&empty)
		return m, m.dispatchWindowSizeMsg()

	case "enter":
		cmdValue := m.cmdInput.GetValue().(string)
		cmdValue = strings.TrimSpace(cmdValue)

		if cmdValue == "q" {
			return m, tea.Quit
		}

		if cmdValue == "" {
			return m, nil
		}

		start, end, err := note.ParseCopyLinesCommand(cmdValue)

		if err != nil {
			m.error = err
			m.noteView.error = err

			return m, tea.Batch(
				m.dispatchWindowSizeMsg(),
				dispatchClearMsg(),
			)
		}

		copiedLines, err := m.store.CopyLines(m.selectedNote.Content, start, end)

		if err != nil {
			m.error = err
			m.noteView.error = err

			return m, tea.Batch(
				m.dispatchWindowSizeMsg(),
				dispatchClearMsg(),
			)
		}

		m.successMessage = fmt.Sprintf(
			"Copied %d %s to clipboard",
			copiedLines,
			utils.Ternary(copiedLines == 1, "line", "lines"),
		)
		m.noteView.successMessage = m.successMessage

		m.cmdMode = false
		empty := ""
		m.cmdInput.Value(&empty)

		return m, tea.Batch(
			m.dispatchWindowSizeMsg(),
			dispatchClearMsg(),
		)
	}

	cmdModel, cmd := m.cmdInput.Update(msg)
	m.cmdInput = cmdModel.(*huh.Input)

	return m, cmd
}
