package markdown

import (
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/styles"
)

// RegisterCatppuccinStyles registers the Catppuccin color themes
func RegisterCatppuccinStyles() {
	// Register Catppuccin Mocha (dark theme)
	// Colors from https://github.com/catppuccin/catppuccin
	styles.Register(chroma.MustNewStyle("catppuccin-mocha", chroma.StyleEntries{
		// Base colors
		chroma.Background:    "#1e1e2e", // Base
		chroma.LineHighlight: "#313244", // Surface0
		chroma.LineNumbers:   "#6c7086", // Overlay0
		chroma.Name:          "#cdd6f4", // Text

		// Syntax elements
		chroma.Comment:            "italic #6c7086", // Overlay0
		chroma.CommentPreproc:     "#f5c2e7",        // Pink
		chroma.Keyword:            "#cba6f7",        // Mauve
		chroma.KeywordType:        "#f38ba8",        // Red
		chroma.KeywordDeclaration: "#cba6f7",        // Mauve
		chroma.Operator:           "#89dceb",        // Sky
		chroma.OperatorWord:       "#cba6f7",        // Mauve
		chroma.Punctuation:        "#cdd6f4",        // Text
		chroma.String:             "#a6e3a1",        // Green
		chroma.StringChar:         "#fab387",        // Peach
		chroma.StringSymbol:       "#f9e2af",        // Yellow
		chroma.Number:             "#fab387",        // Peach
		chroma.NameBuiltin:        "#89b4fa",        // Blue
		chroma.NameBuiltinPseudo:  "#89b4fa",        // Blue
		chroma.NameClass:          "#f9e2af",        // Yellow
		chroma.NameFunction:       "#89b4fa",        // Blue
		chroma.NameVariable:       "#cdd6f4",        // Text
		chroma.NameTag:            "#f38ba8",        // Red
		chroma.NameAttribute:      "#f9e2af",        // Yellow
		chroma.NameDecorator:      "#f5c2e7",        // Pink
		// chroma.LiteralString:      "#a6e3a1",        // Green
		chroma.GenericHeading:    "bold #cdd6f4", // Text bold
		chroma.GenericSubheading: "bold #cdd6f4", // Text bold
		chroma.GenericDeleted:    "#f38ba8",      // Red
		chroma.GenericInserted:   "#a6e3a1",      // Green
		chroma.GenericError:      "#f38ba8",      // Red
		chroma.GenericEmph:       "italic",
		chroma.GenericStrong:     "bold",
		chroma.GenericPrompt:     "#6c7086", // Overlay0
		chroma.Error:             "#f38ba8", // Red
		chroma.CodeLine:          "#f38ba8", // Base
	}))

	// Register Catppuccin Latte (light theme)
	styles.Register(chroma.MustNewStyle("catppuccin-latte", chroma.StyleEntries{
		// Base colors
		chroma.Background:    "#eff1f5", // Base
		chroma.LineHighlight: "#ccd0da", // Surface0
		chroma.LineNumbers:   "#8c8fa1", // Overlay0
		chroma.Name:          "#4c4f69", // Text

		// Syntax elements
		chroma.Comment:            "italic #8c8fa1", // Overlay0
		chroma.CommentPreproc:     "#ea76cb",        // Pink
		chroma.Keyword:            "#8839ef",        // Mauve
		chroma.KeywordType:        "#d20f39",        // Red
		chroma.KeywordDeclaration: "#8839ef",        // Mauve
		chroma.Operator:           "#04a5e5",        // Sky
		chroma.OperatorWord:       "#8839ef",        // Mauve
		chroma.Punctuation:        "#4c4f69",        // Text
		chroma.String:             "#40a02b",        // Green
		chroma.StringChar:         "#fe640b",        // Peach
		chroma.StringSymbol:       "#df8e1d",        // Yellow
		chroma.Number:             "#fe640b",        // Peach
		chroma.NameBuiltin:        "#1e66f5",        // Blue
		chroma.NameBuiltinPseudo:  "#1e66f5",        // Blue
		chroma.NameClass:          "#df8e1d",        // Yellow
		chroma.NameFunction:       "#1e66f5",        // Blue
		chroma.NameVariable:       "#4c4f69",        // Text
		chroma.NameTag:            "#d20f39",        // Red
		chroma.NameAttribute:      "#df8e1d",        // Yellow
		chroma.NameDecorator:      "#ea76cb",        // Pink
		// chroma.LiteralString:      "#40a02b",        // Green
		chroma.GenericHeading:    "bold #4c4f69", // Text bold
		chroma.GenericSubheading: "bold #4c4f69", // Text bold
		chroma.GenericDeleted:    "#d20f39",      // Red
		chroma.GenericInserted:   "#40a02b",      // Green
		chroma.GenericError:      "#d20f39",      // Red
		chroma.GenericEmph:       "italic",
		chroma.GenericStrong:     "bold",
		chroma.GenericPrompt:     "#8c8fa1", // Overlay0
		chroma.Error:             "#d20f39", // Red
	}))
}

// SetCatppuccinTheme sets the Catppuccin theme based on the provided mode
// mode can be "dark" or "light"
func (m *Model) SetCatppuccinTheme(mode string) {
	// Make sure styles are registered
	if styles.Get("catppuccin-mocha") == nil {
		RegisterCatppuccinStyles()
	}

	// Set the appropriate style based on mode
	if mode == "dark" {
		m.SetStyle("catppuccin-mocha")
		m.SetTerminalTheme("dark")
	} else if mode == "light" {
		m.SetStyle("catppuccin-latte")
		m.SetTerminalTheme("light")
	}
}

// SetTerminalTheme sets the terminal theme (dark or light)
func (m *Model) SetTerminalTheme(theme string) {
	if theme == "dark" || theme == "light" {
		m.TerminalTheme = theme
	}
}

func (m *Model) SetStyle(styleName string) {
	style := styles.Get(styleName)
	if style != nil {
		m.Style = styleName
		m.ChromaStyle = style
	}
}
