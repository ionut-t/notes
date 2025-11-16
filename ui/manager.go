package ui

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/notes/internal/help"
	"github.com/ionut-t/notes/internal/keymap"
	"github.com/ionut-t/notes/note"
	"github.com/ionut-t/notes/styles"
)

var (
	viewPadding  = lipgloss.NewStyle().Padding(1, 1)
	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Text.GetForeground())
	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Overlay0.
				GetForeground())
	splitViewSeparator      = " "
	splitViewSeparatorWidth = lipgloss.Width(splitViewSeparator)
	minListWidth            = 50
)

type managerView int

const (
	splitView managerView = iota
	listView
	noteView
)

type focusedView int

const (
	listFocused focusedView = iota
	noteFocused
)

type ManagerModel struct {
	store          *note.Store
	list           list.Model
	view           managerView
	focusedView    focusedView
	noteView       NoteModel
	error          error
	help           help.Model
	width, height  int
	successMessage string
	addNote        AddModel
}

func NewManager(store *note.Store) *ManagerModel {
	notes, err := store.LoadNotes()
	if err != nil {
		notes = []note.Note{}
	}

	items := processNotes(notes)

	delegate := list.NewDefaultDelegate()

	delegate.Styles = styles.ListItemStyles()

	m := ManagerModel{
		store:    store,
		list:     list.New(items, delegate, 0, 0),
		help:     help.New(),
		noteView: NewNoteModel(store, 100, 20),
		error:    err,
	}

	m.list.Title = "Notes"

	m.list.Styles = styles.ListStyles()

	m.list.KeyMap = list.KeyMap{
		CursorUp:             keymap.Up,
		CursorDown:           keymap.Down,
		Filter:               keymap.Search,
		AcceptWhileFiltering: keymap.FullScreen,
		CancelWhileFiltering: keymap.Cancel,
	}

	m.list.FilterInput.PromptStyle = styles.Accent
	m.list.FilterInput.Cursor.Style = styles.Accent

	m.list.InfiniteScrolling = true
	m.list.SetShowHelp(false)

	m.help.Keys.FullHelpBindings = []key.Binding{
		keymap.Up,
		keymap.Down,
		keymap.Left,
		keymap.Right,
		keymap.FullScreen,
		keymap.ExternalEditor,
		keymap.New,
		keymap.Search,
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

func (m ManagerModel) Init() tea.Cmd {
	return tea.SetWindowTitle("Notes")
}

func (m ManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case help.FullViewToggledMsg:
		return m, m.dispatchWindowSizeMsg()

	case editorClosedMsg:
		return m.handleEditorClose(false)

	case noteAddedMsg:
		return m.handleEditorClose(true)

	case cmdNoteDeletedMsg:
		m.list.RemoveItem(m.list.Index())
		if item, ok := m.list.SelectedItem().(item); ok {
			m.store.SetCurrentNoteName(item.title)
			m.noteView.render()
		}

	case cmdSuccessMsg:
		m.successMessage = string(msg)
		m.noteView.successMessage = string(msg)
		return m, tea.Batch(
			dispatchClearMsg(),
			m.dispatchWindowSizeMsg(),
		)

	case cmdErrorMsg:
		m.error = msg
		m.noteView.error = msg
		return m, tea.Sequence(
			dispatchClearMsg(),
			m.dispatchWindowSizeMsg(),
		)

	case cmdNoteRenamedMsg:
		note := msg.note
		m.list.SetItem(m.list.Index(), item{
			title: note.Name,
			desc:  fmt.Sprintf("Last modified: %s", note.UpdatedAt.Format("02/01/2006 15:04")),
		})

	case clearMsg:
		m.successMessage = ""
		m.noteView.successMessage = ""
		m.error = nil
		m.noteView.error = nil

	case changesDiscardedMsg:
		if m.view == splitView {
			m.focusedView = listFocused
		}

	case editor.SaveMsg:
		err := m.store.UpdateCurrentNoteContent(string(msg.Content))
		if err != nil {
			m.error = fmt.Errorf("failed to save note: %w", err)
			m.successMessage = ""
		} else {
			m.successMessage = "Note saved"
			m.error = nil
			m.noteView.updateContent()

			if m.view == splitView {
				m.list.SetItems(processNotes(m.store.GetNotes()))
				m.list.ResetSelected()
			}

			return m, dispatchClearMsg()
		}

	case editor.QuitMsg:
		return m, tea.Quit

	case editor.ErrorMsg:
		return m, m.noteView.dispatchEditorError(msg.Error)

	case tea.KeyMsg:
		if key.Matches(msg, keymap.ForceQuit) {
			return m, tea.Quit
		}

		if m.list.FilterState() == list.Filtering || m.addNote.active {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit):
			return m.handleQuit()

		case key.Matches(msg, keymap.FullScreen):
			return m.handleFullScreen()

		case key.Matches(msg, keymap.ToggleEdit):
			if !m.noteView.isEditing() {
				m.noteView.toggleEdit()
			}

		case key.Matches(msg, keymap.ExternalEditor):
			if ok, cmd := m.triggerNoteEditor(); ok {
				return m, cmd
			}

		case key.Matches(msg, keymap.ChangeFocused):
			if m.view == splitView && !m.noteView.isEditing() {
				if m.noteView.hasChanges() {
					m.noteView.confirm(true)
					break
				}

				if m.focusedView == listFocused {
					m.focusedView = noteFocused
					cmd := m.noteView.focus()
					cmds = append(cmds, cmd)
				} else {
					m.focusedView = listFocused
					m.noteView.blur()
				}
			}

		case key.Matches(msg, keymap.New):
			if m.noteView.isEditing() {
				break
			}

			m.addNote = NewAddModel(m.store)
			m.addNote.height = m.height
			m.addNote.width = m.width
			m.addNote.markAsIntegrated()
			return m, m.addNote.blink()
		}
	}

	if !m.addNote.active {
		switch m.focusedView {
		case listFocused:
			var cmd tea.Cmd

			if !m.help.FullView {
				m.list, cmd = m.list.Update(msg)
				cmds = append(cmds, cmd)
			}

			var selected string

			filteredItems := m.list.VisibleItems()

			if len(filteredItems) > 0 {
				if item, ok := filteredItems[0].(item); ok {
					selected = item.title
				}
			}

			if item, ok := m.list.SelectedItem().(item); ok {
				selected = item.title
			}

			m.store.SetCurrentNoteName(selected)
			width, height := m.getAvailableSizes()
			m.noteView.setSize(width-min(width/2, minListWidth), height)
			m.noteView.updateContent()

		case noteFocused:
			if !m.help.FullView {
				noteViewModel, cmd := m.noteView.Update(msg)
				m.noteView = noteViewModel.(NoteModel)
				cmds = append(cmds, cmd)
			}
		}

		if m.view != noteView {
			helpModel, cmd := m.help.Update(msg)
			m.help = helpModel.(help.Model)
			cmds = append(cmds, cmd)
		}
	}

	if m.addNote.active {
		addNoteModel, cmd := m.addNote.Update(msg)
		m.addNote = addNoteModel.(AddModel)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m ManagerModel) View() string {
	if m.addNote.active {
		return m.addNote.View()
	}

	switch m.view {
	case listView:
		return viewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case noteView:
		return m.noteView.View()

	case splitView:
		return m.getSplitView()

	default:
		return ""
	}
}

func (m ManagerModel) getSplitView() string {
	horizontalFrameSize := viewPadding.GetHorizontalFrameSize()
	horizontalFrameBorderSize := activeBorder.GetHorizontalFrameSize()

	availableWidth := m.width - horizontalFrameSize

	listWidth := min(minListWidth, availableWidth/2) - horizontalFrameBorderSize*2 - splitViewSeparatorWidth
	noteWidth := availableWidth - listWidth - horizontalFrameBorderSize*2 - splitViewSeparatorWidth

	var joinedContent string

	if m.focusedView == listFocused {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			activeBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			inactiveBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.noteView.View()),
		)
	} else {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			inactiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			activeBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.noteView.View()),
		)
	}

	renderedView := viewPadding.Render(joinedContent)

	return renderedView + "\n" + m.statusBarView()
}

func (m ManagerModel) getViewInCmdMode() string {
	return viewPadding.Render(m.list.View()) + "\n" + m.statusBarView()
}

func (m ManagerModel) statusBarView() string {
	if m.error != nil {
		return styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	if m.list.FilterState() == list.Filtering {
		m.help.Keys.ShortHelpBindings = []key.Binding{
			keymap.Cancel,
		}
	} else {
		m.help.Keys.ShortHelpBindings = []key.Binding{
			keymap.FullScreen,
			keymap.ExternalEditor,
			keymap.Search,
			keymap.New,
			keymap.Quit,
			keymap.Help,
		}
	}

	if m.help.FullView {
		return m.help.View()
	}

	return lipgloss.NewStyle().Margin(0, 2).Render(m.help.View())
}

func processNotes(notes []note.Note) []list.Item {
	items := make([]list.Item, len(notes))

	for i, n := range notes {
		items[i] = item{
			title: n.Name,
			desc:  fmt.Sprintf("Last modified: %s", n.UpdatedAt.Format("02/01/2006 15:04")),
		}
	}

	return items
}

func (m *ManagerModel) handleWindowSize(msg tea.WindowSizeMsg) {
	if msg.Width < 2*minListWidth {
		switch m.view {
		case splitView:
			m.view = listView
		case listView:
			m.view = splitView
		}
	}

	m.width, m.height = msg.Width, msg.Height

	availableWidth, availableHeight := m.getAvailableSizes()

	m.help.SetSize(msg.Width, msg.Height)

	if m.view == listView {
		m.list.SetSize(availableWidth, availableHeight)
		m.help.SetSize(msg.Width, msg.Height)
	}

	if m.view == noteView {
		m.noteView.setSize(msg.Width, msg.Height)
	}

	if m.view == splitView {
		listWidth := min(availableWidth/2, minListWidth)

		// Set list dimensions
		m.list.SetHeight(availableHeight)
		m.list.SetWidth(listWidth)

		// Set note view dimensions
		m.noteView.setSize(availableWidth-listWidth, availableHeight)
	}
}

func (m ManagerModel) handleEditorClose(isNew bool) (ManagerModel, tea.Cmd) {
	notes, err := m.store.LoadNotes()
	if err != nil {
		return m, dispatch(cmdErrorMsg(err))
	}

	m.list.SetItems(processNotes(notes))

	m.noteView.updateContent()

	if _, ok := m.store.GetCurrentNote(); ok {
		// there seems to be a bug in bubbletea that causes the filter to not
		// preserve the selected item after the list is updated
		// reset the filter until a better solution is found
		m.list.ResetFilter()
	}

	if m.store.IsFirstNote() || isNew {
		m.list.ResetSelected()
	}

	return m, tea.Sequence(
		m.dispatchWindowSizeMsg(),
		tea.EnableMouseCellMotion,
	)
}

func (m ManagerModel) handleQuit() (ManagerModel, tea.Cmd) {
	if m.help.FullView {
		m.help.FullView = false
		return m, dispatch(help.FullViewToggledMsg{})
	}

	if m.view == noteView {
		m.view = splitView
		m.focusedView = listFocused
		m.noteView.fullScreen = false
		m.noteView.help.FullView = false
		return m, m.dispatchWindowSizeMsg()
	}

	return m, tea.Quit
}

func (m ManagerModel) handleFullScreen() (ManagerModel, tea.Cmd) {
	if len(m.list.Items()) == 0 ||
		m.noteView.isEditing() {
		return m, nil
	}

	m.noteView.fullScreen = !m.noteView.fullScreen
	m.help.FullView = false
	m.noteView.setSize(m.width, m.height)

	m.noteView.updateContent()

	var cmds []tea.Cmd

	if m.noteView.fullScreen {
		m.view = noteView
		m.focusedView = noteFocused
		cmd := m.noteView.focus()
		cmds = append(cmds, cmd)
	} else {
		m.view = splitView
		m.focusedView = listFocused
		m.noteView.blur()
	}

	cmds = append(cmds, m.dispatchWindowSizeMsg())
	return m, tea.Batch(cmds...)
}

func (m *ManagerModel) triggerNoteEditor() (bool, tea.Cmd) {
	if len(m.list.Items()) == 0 {
		return false, nil
	}

	if note, ok := m.store.GetCurrentNote(); ok {
		notePath := m.store.GetNotePath(note.Name)
		execCmd := tea.ExecProcess(exec.Command(m.store.GetEditor(), notePath), func(error) tea.Msg {
			return editorClosedMsg{}
		})

		return true, execCmd
	}

	return false, nil
}

func (m ManagerModel) dispatchWindowSizeMsg() tea.Cmd {
	return dispatch(tea.WindowSizeMsg{Width: m.width, Height: m.height})
}

func (m ManagerModel) getAvailableSizes() (int, int) {
	h, v := viewPadding.GetFrameSize()

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - activeBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
}
