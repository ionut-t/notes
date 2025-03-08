package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	chStyles "github.com/alecthomas/chroma/styles"
)

type LineType int

const (
	LineTypeNormal LineType = iota
	LineTypeHeader
	LineTypeCodeFence
	LineTypeCode
	LineTypeEmpty
	LineTypeComment
)

// Line represents a single line in the markdown content with metadata
type Line struct {
	Content     string
	Type        LineType
	HeaderLevel int
	CodeLang    string
}

// Model represents the markdown rendering model
type Model struct {
	Content       string
	Width         int
	Lines         []Line
	LineNumbers   bool
	Style         string // Name of the Chroma style to use
	ChromaStyle   *chroma.Style
	ChromaConfig  *ChromaConfig
	DefaultLexer  string // Default lexer to use when language is not specified
	TerminalTheme string // Terminal theme: "dark" or "light"
}

// ChromaConfig holds configuration for Chroma highlighting
type ChromaConfig struct {
	TabWidth      int
	WithClasses   bool
	LineNumbers   bool
	LineNumberBg  string
	LineNumberFg  string
	HighlightLine HighlightFunc
}

// HighlightFunc is a function that determines if a line should be highlighted
type HighlightFunc func(line int) bool

// New creates a new markdown model
func New(content string, width int) Model {
	// Default Chroma configuration
	chromaConfig := &ChromaConfig{
		TabWidth:    4,
		WithClasses: false,
		LineNumbers: false,
	}

	RegisterCatppuccinStyles()

	m := Model{
		Content:       content,
		Width:         width,
		LineNumbers:   false,
		Style:         "catppuccin-mocha",
		ChromaStyle:   chStyles.Get("catppuccin-mocha"),
		ChromaConfig:  chromaConfig,
		DefaultLexer:  "text",
		TerminalTheme: "dark",
	}

	// If Chroma style is nil, fallback to a built-in style
	if m.ChromaStyle == nil {
		m.Style = "monokai"
		m.ChromaStyle = chStyles.Get("monokai")
	}

	m.ParseLines()
	return m
}

func (m *Model) SetLineNumbers(show bool) {
	m.LineNumbers = show
}

// ParseLines parses the content into individual lines with metadata
func (m *Model) ParseLines() {
	contentLines := strings.Split(m.Content, "\n")
	m.Lines = make([]Line, len(contentLines))

	inCodeBlock := false
	var codeLang string

	for i, content := range contentLines {
		line := Line{
			Content: content,
		}

		// Check if line is a code fence
		if strings.HasPrefix(content, "```") {
			line.Type = LineTypeCodeFence
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				codeLang = strings.TrimPrefix(content, "```")
				line.CodeLang = codeLang
			} else {
				// End of code block
				inCodeBlock = false
				codeLang = ""
			}
		} else if inCodeBlock {
			// Line is inside a code block
			line.Type = LineTypeCode
			line.CodeLang = codeLang
		} else if strings.HasPrefix(content, "#") {
			level := 0
			for j, char := range content {
				if char == '#' {
					level++
				} else if char == ' ' && j == level {
					// Proper heading format with space after hash signs
					line.Type = LineTypeHeader
					line.HeaderLevel = level
					line.Content = strings.TrimSpace(content[j:])
					break
				} else {
					// Not a proper heading format - treat as comment
					line.Type = LineTypeComment
					break
				}
			}
		} else if len(strings.TrimSpace(content)) == 0 {
			// Line is empty
			line.Type = LineTypeEmpty
		} else {
			// Line is normal text
			line.Type = LineTypeNormal
		}

		m.Lines[i] = line
	}
}

// formatHeaderLine applies formatting to a header line
func (m *Model) formatHeaderLine(line Line) string {
	formattedLine := m.applyInlineFormatting(line.Content)

	switch line.HeaderLevel {
	case 1:
		formattedLine = reset + bold + blue + formattedLine + reset
	case 2, 3, 4, 5, 6:
		formattedLine = reset + blue + formattedLine + reset
	}

	return formattedLine
}

// applyInlineFormatting applies inline formatting (bold, italic, etc.)
func (m *Model) applyInlineFormatting(text string) string {
	// Links: [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRegex.ReplaceAllString(text, blue+bold+"$1"+reset+" "+blue+"($2)"+reset)

	// Bold: **text** or __text__
	boldRegex1 := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = boldRegex1.ReplaceAllString(text, bold+"$1"+reset)

	boldRegex2 := regexp.MustCompile(`__(.*?)__`)
	text = boldRegex2.ReplaceAllString(text, bold+"$1"+reset)

	// Italic: *text* or _text_
	italicRegex1 := regexp.MustCompile(`\*(.*?)\*`)
	text = italicRegex1.ReplaceAllString(text, italic+"$1"+reset)

	italicRegex2 := regexp.MustCompile(`_(.*?)_`)
	text = italicRegex2.ReplaceAllString(text, italic+"$1"+reset)

	// Inline code: `code`
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllString(text, cyan+"$1"+reset)

	return strings.TrimSpace(text)
}

// syntaxHighlightWithChroma uses Chroma to highlight code
func (m *Model) syntaxHighlightWithChroma(code, language string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}

	// Normalize language name
	normalizedLang := strings.ToLower(strings.TrimSpace(language))

	// Get the appropriate lexer
	lexer := lexers.Get(normalizedLang)
	if lexer == nil {
		// Try to get lexer by analyzing the code
		lexer = lexers.Analyse(code)
		if lexer == nil {
			// Fallback to default lexer
			lexer = lexers.Get(m.DefaultLexer)
			if lexer == nil {
				// Last resort fallback
				lexer = lexers.Fallback
			}
		}
	}

	// Use a simple formatter that works with terminal output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Choose appropriate style based on terminal theme
	style := m.ChromaStyle
	if style == nil {
		if m.TerminalTheme == "light" {
			style = chStyles.Get("github")
		} else {
			style = chStyles.Get("catppuccin-mocha")
		}
		if style == nil {
			style = chStyles.Fallback
		}
	}

	// Create a buffer to hold the highlighted code
	var buf strings.Builder

	// Get the tokens from the lexer
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Fall back to unhighlighted code on error
	}

	// Format the tokens
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code // Fall back to unhighlighted code on error
	}

	return buf.String()
}

// highlightCodeBlock highlights an entire code block using Chroma
func (m *Model) highlightCodeBlock(block []Line) []string {
	if len(block) == 0 {
		return []string{}
	}

	// Extract language from the first line
	language := block[0].CodeLang

	// Combine all lines of code
	var codeBuilder strings.Builder
	for _, line := range block {
		codeBuilder.WriteString(line.Content)
		codeBuilder.WriteString("\n")
	}
	code := codeBuilder.String()

	// Highlight the code
	highlighted := m.syntaxHighlightWithChroma(code, language)

	// Split the highlighted code back into lines
	highlightedLines := strings.Split(highlighted, "\n")

	// Handle potential differences in line count due to formatting
	// (ensure we have at least as many lines as the original code block)
	if len(highlightedLines) < len(block) {
		for i := len(highlightedLines); i < len(block); i++ {
			highlightedLines = append(highlightedLines, "")
		}
	}

	return highlightedLines
}

// addLineNumber adds line number to the beginning of a line
func (m *Model) addLineNumber(lineNum int, line string) string {
	if !m.LineNumbers {
		return line
	}

	// Format line number with right alignment and padding
	lineNumStr := fmt.Sprintf("%3d ", lineNum)
	return gray + lineNumStr + reset + line
}

// wrapLine wraps a line to fit within the specified width
func (m *Model) wrapLine(line string, width int) []string {
	// If line doesn't need wrapping, return as is
	if len(line) <= width {
		return []string{line}
	}

	var wrappedLines []string
	remainingLine := line

	for len(remainingLine) > width {
		// Find a good breaking point
		breakPoint := width
		for breakPoint > 0 && !strings.Contains(" \t-,.:;!?)", string(remainingLine[breakPoint-1])) {
			breakPoint--
		}

		// If no good breaking point found, force break at width
		if breakPoint == 0 {
			breakPoint = width
		}

		wrappedLines = append(wrappedLines, remainingLine[:breakPoint])
		remainingLine = remainingLine[breakPoint:] // No indentation for wrapped lines
	}

	if len(remainingLine) > 0 {
		wrappedLines = append(wrappedLines, remainingLine)
	}

	return wrappedLines
}

// Render renders the markdown content
func (m *Model) Render() string {
	var result strings.Builder

	// collect code blocks for Chroma highlighting
	var codeBlock []Line
	inCodeBlock := false

	for i := 0; i < len(m.Lines); i++ {
		line := m.Lines[i]
		lineNum := i + 1

		if line.Type == LineTypeCodeFence {
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				codeBlock = []Line{}
			} else {
				// End of code block - highlight and add to result
				highlightedLines := m.highlightCodeBlock(codeBlock)

				for j, hLine := range highlightedLines {
					if j < len(codeBlock) { // Safety check
						codeLineNum := i - len(codeBlock) + j + 1
						lineWithNum := m.addLineNumber(codeLineNum, "  "+hLine)
						result.WriteString(lineWithNum + "\n")
					}
				}

				inCodeBlock = false
				codeBlock = nil
			}
			// Don't render the code fence markers
			continue
		}

		if inCodeBlock {
			// Collect line for later highlighting
			codeBlock = append(codeBlock, line)
			continue
		}

		// Process non-code-block lines
		var formattedLine string

		switch line.Type {
		case LineTypeHeader:
			// Render header with appropriate styling
			formattedLine = m.formatHeaderLine(line)

		case LineTypeEmpty:
			// Render empty line
			formattedLine = ""

		case LineTypeComment:
			formattedLine = dimmed + gray + line.Content + reset

		default:
			// Render normal text with inline formatting
			formattedLine = m.applyInlineFormatting(line.Content)
		}

		// For normal text (not code or comments), wrap the line if it's too long
		if line.Type != LineTypeCode && line.Type != LineTypeComment && len(formattedLine) > 0 {
			// Calculate available width accounting for line numbers
			// 5 = 3 digits for line number + 1 space + 1 padding
			availableWidth := m.Width - 5

			if len(formattedLine) > availableWidth {
				wrappedLines := m.wrapLine(formattedLine, availableWidth)

				// Add the first line with line number
				lineWithNum := m.addLineNumber(lineNum, wrappedLines[0])
				result.WriteString(lineWithNum + "\n")

				// Add continuation lines with no line number
				for j := 1; j < len(wrappedLines); j++ {
					if m.LineNumbers {
						result.WriteString("    " + wrappedLines[j] + "\n")
					} else {
						result.WriteString(wrappedLines[j] + "\n")
					}
				}

				continue // Skip the normal line addition below since we've handled it
			}
		}

		lineWithNum := m.addLineNumber(lineNum, formattedLine)
		result.WriteString(lineWithNum + "\n")
	}

	return result.String()
}

// RenderPreservingAll renders the markdown content preserving every line
func (m *Model) RenderPreservingAll() string {
	var result strings.Builder

	// collect code blocks for Chroma highlighting
	var codeBlock []Line
	inCodeBlock := false

	for i := 0; i < len(m.Lines); i++ {
		line := m.Lines[i]
		lineNum := i + 1

		if line.Type == LineTypeCodeFence {
			// Handle code fence markers
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				codeBlock = []Line{}
				// Output the code fence marker
				formattedLine := m.addLineNumber(lineNum, line.Content)
				result.WriteString(formattedLine + "\n")
			} else {
				// End of code block - highlight and add to result
				highlightedLines := m.highlightCodeBlock(codeBlock)

				for j, hLine := range highlightedLines {
					if j < len(codeBlock) { // Safety check
						codeLineNum := i - len(codeBlock) + j
						lineWithNum := m.addLineNumber(codeLineNum, "  "+hLine)
						result.WriteString(lineWithNum + "\n")
					}
				}

				// Output the end code fence marker
				formattedLine := m.addLineNumber(lineNum, line.Content)
				result.WriteString(formattedLine + "\n")

				inCodeBlock = false
				codeBlock = nil
			}
			continue
		}

		if inCodeBlock {
			// Collect line for later highlighting
			codeBlock = append(codeBlock, line)
			continue
		}

		// Process non-code-block lines
		var formattedLine string

		switch line.Type {
		case LineTypeHeader:
			// Render header with appropriate styling
			formattedLine = m.formatHeaderLine(line)

		case LineTypeEmpty:
			// Render empty line
			formattedLine = ""

		case LineTypeComment:
			// Render comments with dimmed color and no formatting
			formattedLine = dimmed + gray + line.Content + reset

		default:
			// Render normal text with inline formatting
			formattedLine = m.applyInlineFormatting(line.Content)
		}

		// For normal text, wrap the line if it's too long
		if line.Type != LineTypeComment && len(formattedLine) > 0 {
			// Calculate available width accounting for line numbers
			availableWidth := m.Width - 5

			if len(formattedLine) > availableWidth {
				wrappedLines := m.wrapLine(formattedLine, availableWidth)

				// Add the first line with line number
				lineWithNum := m.addLineNumber(lineNum, wrappedLines[0])
				result.WriteString(lineWithNum + "\n")

				// Add continuation lines with indentation
				for j := 1; j < len(wrappedLines); j++ {
					if m.LineNumbers {
						result.WriteString(gray + "    " + reset + wrappedLines[j] + "\n")
					} else {
						result.WriteString(wrappedLines[j] + "\n")
					}
				}

				continue // Skip the normal line addition below
			}
		}

		lineWithNum := m.addLineNumber(lineNum, formattedLine)
		result.WriteString(lineWithNum + "\n")
	}

	return result.String()
}
