package keymap

import (
	"reflect"

	"github.com/charmbracelet/bubbles/key"
)

var Up = key.NewBinding(
	key.WithKeys("up", "k"),
	key.WithHelp("↑ / k", "up"),
)

var Down = key.NewBinding(
	key.WithKeys("down", "j"),
	key.WithHelp("↓ / j", "down"),
)

var Left = key.NewBinding(
	key.WithKeys("left", "h"),
	key.WithHelp("← / h", "left"),
)

var Right = key.NewBinding(
	key.WithKeys("right", "l"),
	key.WithHelp("→ / l", "right"),
)

var Select = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "full screen"),
)

var NewLine = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "new line"),
)

var Back = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "back"),
)

var Cancel = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "cancel"),
)

var Quit = key.NewBinding(
	key.WithKeys("esc", "q"),
	key.WithHelp("esc / q", "quit"),
)

var QuitForm = key.NewBinding(
	key.WithKeys("ctrl+c", "esc"),
	key.WithHelp("ctrl+c / esc", "quit"),
)

var ForceQuit = key.NewBinding(
	key.WithKeys("ctrl+c"),
	key.WithHelp("ctrl+c", "quit"),
)

var Help = key.NewBinding(
	key.WithKeys("?"),
	key.WithHelp("?", "help"),
)

var Search = key.NewBinding(
	key.WithKeys("/"),
	key.WithHelp("/", "search"),
)

var ExitSearch = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "exit search"),
)

var Editor = key.NewBinding(
	key.WithKeys("ctrl+e"),
	key.WithHelp("ctrl+e", "open editor"),
)

var QuickEditor = key.NewBinding(
	key.WithKeys("e"),
	key.WithHelp("e", "open in editor"),
)

var Continue = key.NewBinding(
	key.WithKeys("alt+enter", "ctrl+s"),
	key.WithHelp("alt+enter / ctrl+s", "continue"),
)

var Save = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "save"),
)

var Rename = key.NewBinding(
	key.WithKeys("r"),
	key.WithHelp("r", "rename"),
)

var Delete = key.NewBinding(
	key.WithKeys("ctrl+d"),
	key.WithHelp("ctrl+d", "delete"),
)

var Copy = key.NewBinding(
	key.WithKeys("c"),
	key.WithHelp("c", "copy note to clipboard"),
)

var CopyLines = key.NewBinding(
	key.WithKeys(":"),
	key.WithHelp(":co", `copy lines to clipboard
						 Examples:
						 co 1 (copy line 1)
		 				 co 1 2 (copy lines 1 to 2)
		 				 co 1 1 (copy line 1)
						 co 20 > 2 (copy lines 20 to 22)
						 co 20 < 2 (copy lines 18 to 20)
						 co 20 > -1 (copy lines 20 to the end)
						 co 20 < -1 (copy lines 1 to 20)`,
	),
)

var Command = key.NewBinding(
	key.WithKeys(":"),
	key.WithHelp(":", "enter command"),
)

var VLine = key.NewBinding(
	key.WithKeys("V"),
	key.WithHelp("V", "toggle line numbers"),
)

var New = key.NewBinding(
	key.WithKeys("n"),
	key.WithHelp("n", "new note"),
)

var Accept = key.NewBinding(
	key.WithKeys("y", "Y"),
	key.WithHelp("y", "yes"),
)

var Reject = key.NewBinding(
	key.WithKeys("n", "N"),
	key.WithHelp("n", "no"),
)

type Model struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Help       key.Binding
	Quit       key.Binding
	Back       key.Binding
	Select     key.Binding
	Search     key.Binding
	ExitSearch key.Binding
	Copy       key.Binding

	ShortHelpBindings []key.Binding
	FullHelpBindings  []key.Binding
}

func (k Model) ShortHelp() []key.Binding {
	return k.ShortHelpBindings
}

func (k Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func CombineKeys(a, b Model) Model {
	result := Model{}

	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)
	resultVal := reflect.ValueOf(&result).Elem()

	for i := 0; i < resultVal.NumField(); i++ {
		field := resultVal.Type().Field(i)

		// Try to get value from b first, if it's not zero value, use it
		bField := bVal.FieldByName(field.Name)
		if !bField.IsZero() {
			resultVal.Field(i).Set(bField)
			continue
		}

		// If b's field is zero value, use a's field
		aField := aVal.FieldByName(field.Name)
		if !aField.IsZero() {
			resultVal.Field(i).Set(aField)
		}
	}

	return result
}

func ReplaceBinding(bindings []key.Binding, newBinding key.Binding) []key.Binding {
	for i, binding := range bindings {
		if binding.Help().Key == newBinding.Help().Key {
			bindings[i] = newBinding
		}
	}

	return bindings
}

var DefaultKeyMap = Model{
	Up:     Up,
	Down:   Down,
	Left:   Left,
	Right:  Right,
	Search: Search,
	Back:   Back,
	Help:   Help,
	Quit:   Quit,
}

var SearchKeyMap = Model{
	ExitSearch: ExitSearch,
	Quit:       Quit,
}

var ListKeyMap = Model{
	Up:     Up,
	Down:   Down,
	Select: Select,
}
