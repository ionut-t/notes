package markdown

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	chStyles "github.com/alecthomas/chroma/styles"
	"github.com/ionut-t/notes/styles"
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

type Model struct {
	Content       string
	Width         int
	Lines         []Line
	LineNumbers   bool
	Style         string // Name of the Chroma style to use
	ChromaStyle   *chroma.Style
	DefaultLexer  string // Default lexer to use when language is not specified
	TerminalTheme string // Terminal theme: "dark" or "light"
}

// New creates a new markdown model
func New(content string, width int) Model {
	RegisterCatppuccinStyles()

	m := Model{
		Content:       content,
		Width:         width,
		LineNumbers:   false,
		Style:         "catppuccin-mocha",
		ChromaStyle:   chStyles.Get("catppuccin-mocha"),
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

		// check if line is a code fence
		if strings.HasPrefix(content, "```") {
			line.Type = LineTypeCodeFence
			if !inCodeBlock {
				// start of code block
				inCodeBlock = true
				codeLang = strings.TrimPrefix(content, "```")
				line.CodeLang = codeLang
			} else {
				// end of code block
				inCodeBlock = false
				codeLang = ""
			}
		} else if inCodeBlock {
			// line is inside a code block
			line.Type = LineTypeCode
			line.CodeLang = codeLang
		} else if strings.HasPrefix(content, "#") {
			level := 0
			for j, char := range content {
				if char == '#' {
					level++
				} else if char == ' ' && j == level {
					// proper heading format with space after hash signs
					line.Type = LineTypeHeader
					line.HeaderLevel = level
					line.Content = strings.TrimSpace(content[j:])
					break
				} else {
					// not a proper heading format - treat as comment
					line.Type = LineTypeComment
					break
				}
			}
		} else if len(strings.TrimSpace(content)) == 0 {
			// line is empty
			line.Type = LineTypeEmpty
		} else {
			// line is normal text
			line.Type = LineTypeNormal
		}

		m.Lines[i] = line
	}
}

// formatHeaderLine applies formatting to a header line
func (m *Model) formatHeaderLine(line Line) string {
	// style the header content with lipgloss
	content := line.Content

	// apply inline formatting for header content
	content = m.applyInlineFormatting(content)

	// apply header styling based on level
	switch line.HeaderLevel {
	case 1:
		return styles.Primary.Bold(true).Render(content)
	case 2:
		return styles.Info.Render(content)
	case 3, 4, 5, 6:
		return styles.Accent.Italic(true).Render(content)
	default:
		return content
	}
}

// applyInlineFormatting applies inline formatting
func (m *Model) applyInlineFormatting(text string) string {
	// links: [text](url)
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := linkRegex.FindStringSubmatch(match)
		if len(parts) == 3 {
			linkText := parts[1]
			url := parts[2]
			return styles.Info.Bold(true).Render(linkText) + " " + styles.Info.Render("("+url+")")
		}
		return match
	})

	// bold: **text** or __text__
	boldRegex1 := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = boldRegex1.ReplaceAllStringFunc(text, func(match string) string {
		parts := boldRegex1.FindStringSubmatch(match)
		if len(parts) == 2 {
			return styles.Text.Bold(true).Render(parts[1])
		}
		return match
	})

	boldRegex2 := regexp.MustCompile(`__(.*?)__`)
	text = boldRegex2.ReplaceAllStringFunc(text, func(match string) string {
		parts := boldRegex2.FindStringSubmatch(match)
		if len(parts) == 2 {
			return styles.Text.Bold(true).Render(parts[1])
		}
		return match
	})

	// italic: *text* or _text_
	italicRegex1 := regexp.MustCompile(`\*(.*?)\*`)
	text = italicRegex1.ReplaceAllStringFunc(text, func(match string) string {
		parts := italicRegex1.FindStringSubmatch(match)
		if len(parts) == 2 {
			return styles.Text.Italic(true).Render(parts[1])
		}
		return match
	})

	italicRegex2 := regexp.MustCompile(`_(.*?)_`)
	text = italicRegex2.ReplaceAllStringFunc(text, func(match string) string {
		parts := italicRegex2.FindStringSubmatch(match)
		if len(parts) == 2 {
			return styles.Text.Italic(true).Render(parts[1])
		}
		return match
	})

	// inline code: `code`
	codeRegex := regexp.MustCompile("`([^`]+)`")
	text = codeRegex.ReplaceAllStringFunc(text, func(match string) string {
		parts := codeRegex.FindStringSubmatch(match)
		if len(parts) == 2 {
			return styles.Accent.Render(parts[1])
		}
		return match
	})

	return text
}

// syntaxHighlightWithChroma uses Chroma to highlight code
func (m *Model) syntaxHighlightWithChroma(code, language string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}

	normalizedLang := strings.ToLower(strings.TrimSpace(language))

	lexer := lexers.Get(normalizedLang)
	if lexer == nil {
		// try to get lexer by analyzing the code
		lexer = lexers.Analyse(code)
		if lexer == nil {
			lexer = lexers.Get(m.DefaultLexer)
			if lexer == nil {
				lexer = lexers.Fallback
			}
		}
	}

	formatter := formatters.Get("terminal256")

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

	// create a buffer to hold the highlighted code
	var buf strings.Builder

	// get the tokens from the lexer
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code
	}

	return buf.String()
}

// highlightCodeBlock highlights an entire code block using Chroma
func (m *Model) highlightCodeBlock(block []Line) []string {
	if len(block) == 0 {
		return []string{}
	}

	// extract language from the first line
	language := block[0].CodeLang

	// combine all lines of code
	var codeBuilder strings.Builder
	for _, line := range block {
		codeBuilder.WriteString(line.Content)
		codeBuilder.WriteString("\n")
	}
	code := codeBuilder.String()

	highlighted := m.syntaxHighlightWithChroma(code, language)

	// split the highlighted code back into lines
	highlightedLines := strings.Split(highlighted, "\n")

	// handle potential differences in line count due to formatting
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

	// format line number with right alignment and padding
	lineNumStr := fmt.Sprintf("%3d ", lineNum)
	return styles.Subtext0.Render(lineNumStr) + line
}

// estimateVisibleLength estimates the visible length of text with lipgloss styling
func (m *Model) estimateVisibleLength(text string) int {
	// count visible runes and ignore ANSI escape sequences
	// Lipgloss uses ANSI escape sequences which start with ESC (27) and '['
	// and end with 'm'
	visible := 0
	inEscapeSeq := false

	for _, r := range text {
		if inEscapeSeq {
			if r == 'm' {
				inEscapeSeq = false
			}
			continue
		}

		if r == 27 { // ESC character
			inEscapeSeq = true
			continue
		}

		visible++
	}

	return visible
}

// wrapLine wraps a line to fit within the specified width
func (m *Model) wrapLine(line string, width int) []string {
	if m.estimateVisibleLength(line) <= width {
		return []string{line}
	}

	// split the styled text into words and try to respect styling
	var wrappedLines []string
	words := strings.Split(line, " ")

	currentLine := ""
	currentLineVisibleLength := 0

	for _, word := range words {
		wordVisibleLength := m.estimateVisibleLength(word)

		// if adding this word would exceed width, start a new line
		if currentLineVisibleLength > 0 &&
			currentLineVisibleLength+1+wordVisibleLength > width {
			wrappedLines = append(wrappedLines, currentLine)
			currentLine = word
			currentLineVisibleLength = wordVisibleLength
		} else {
			if currentLineVisibleLength > 0 {
				currentLine += " "
				currentLineVisibleLength++
			}
			currentLine += word
			currentLineVisibleLength += wordVisibleLength
		}
	}

	// add the last line
	if currentLine != "" {
		wrappedLines = append(wrappedLines, currentLine)
	}

	return wrappedLines
}

// Render renders the markdown content
func (m *Model) Render() string {
	var result strings.Builder

	// collect code blocks for Chroma highlighting
	var codeBlock []Line
	inCodeBlock := false

	for i, line := range m.Lines {
		lineNum := i + 1

		if line.Type == LineTypeCodeFence {
			if !inCodeBlock {
				// start of code block
				inCodeBlock = true
				codeBlock = []Line{}
			} else {
				// end of code block - highlight and add to result
				highlightedLines := m.highlightCodeBlock(codeBlock)

				for j, hLine := range highlightedLines {
					if j < len(codeBlock) {
						codeLineNum := i - len(codeBlock) + j + 1
						lineWithNum := m.addLineNumber(codeLineNum, "  "+hLine)
						result.WriteString(lineWithNum + "\n")
					}
				}

				inCodeBlock = false
				codeBlock = nil
			}
			// don't render the code fence markers
			continue
		}

		if inCodeBlock {
			// collect line for later highlighting
			codeBlock = append(codeBlock, line)
			continue
		}

		// process non-code-block lines
		var formattedLine string

		switch line.Type {
		case LineTypeHeader:
			formattedLine = m.formatHeaderLine(line)

		case LineTypeEmpty:
			formattedLine = ""

		case LineTypeComment:
			formattedLine = styles.Subtext0.Faint(true).Render(line.Content)

		default:
			formattedLine = m.applyInlineFormatting(line.Content)
		}

		// for normal text (not code or comments), wrap the line if it's too long
		if line.Type != LineTypeCode && line.Type != LineTypeComment && len(formattedLine) > 0 {
			// Calculate available width accounting for line numbers
			availableWidth := m.Width
			if m.LineNumbers {
				availableWidth -= 5
			}

			visibleLength := m.estimateVisibleLength(formattedLine)
			if visibleLength > availableWidth {
				wrappedLines := m.wrapLine(formattedLine, availableWidth)

				// add the first line with line number
				lineWithNum := m.addLineNumber(lineNum, wrappedLines[0])
				result.WriteString(lineWithNum + "\n")

				// add continuation lines with no line number
				for j := 1; j < len(wrappedLines); j++ {
					if m.LineNumbers {
						// create a continuation indicator with subtle styling
						continuationPrefix := styles.Subtext0.Render("    ")
						result.WriteString(continuationPrefix + wrappedLines[j] + "\n")
					} else {
						result.WriteString(wrappedLines[j] + "\n")
					}
				}

				continue // skip the normal line addition below
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

	for i, line := range m.Lines {
		lineNum := i + 1

		if line.Type == LineTypeCodeFence {
			// handle code fence markers
			if !inCodeBlock {
				// start of code block
				inCodeBlock = true
				codeBlock = []Line{}
				// output the code fence marker
				formattedLine := m.addLineNumber(lineNum, styles.Subtext1.Render(line.Content))
				result.WriteString(formattedLine + "\n")
			} else {
				// end of code block - highlight and add to result
				highlightedLines := m.highlightCodeBlock(codeBlock)

				for j, hLine := range highlightedLines {
					if j < len(codeBlock) {
						codeLineNum := i - len(codeBlock) + j
						lineWithNum := m.addLineNumber(codeLineNum, "  "+hLine)
						result.WriteString(lineWithNum + "\n")
					}
				}

				formattedLine := m.addLineNumber(lineNum, styles.Subtext1.Render(line.Content))
				result.WriteString(formattedLine + "\n")

				inCodeBlock = false
				codeBlock = nil
			}
			continue
		}

		if inCodeBlock {
			codeBlock = append(codeBlock, line)
			continue
		}

		var formattedLine string

		switch line.Type {
		case LineTypeHeader:
			formattedLine = m.formatHeaderLine(line)

		case LineTypeEmpty:
			formattedLine = ""

		case LineTypeComment:
			formattedLine = styles.Subtext0.Faint(true).Render(line.Content)

		default:
			formattedLine = m.applyInlineFormatting(line.Content)
		}

		// for normal text, wrap the line if it's too long
		if line.Type != LineTypeComment && len(formattedLine) > 0 {
			// calculate available width accounting for line numbers
			availableWidth := m.Width
			if m.LineNumbers {
				availableWidth -= 5
			}

			visibleLength := m.estimateVisibleLength(formattedLine)
			if visibleLength > availableWidth {
				wrappedLines := m.wrapLine(formattedLine, availableWidth)

				// add the first line with line number
				lineWithNum := m.addLineNumber(lineNum, wrappedLines[0])
				result.WriteString(lineWithNum + "\n")

				// add continuation lines with indentation
				for j := 1; j < len(wrappedLines); j++ {
					if m.LineNumbers {
						continuationPrefix := styles.Subtext0.Render("    ")
						result.WriteString(continuationPrefix + wrappedLines[j] + "\n")
					} else {
						result.WriteString(wrappedLines[j] + "\n")
					}
				}

				continue // skip the normal line addition below
			}
		}

		lineWithNum := m.addLineNumber(lineNum, formattedLine)
		result.WriteString(lineWithNum + "\n")
	}

	return result.String()
}
