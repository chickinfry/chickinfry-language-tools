package chiklang

import (
	"sort"
	"strings"
)

// Format rewrites a parsed file into the canonical layout.
//
// The separators are the Go style the decision fixed:
//   - a line break ends an entry inside a block, so a semicolon is dropped
//   - a construct that spans lines is broken inside { }, ( ), or [ ]
//   - items on one line separate with commas
//   - a trailing comma is kept on a multi-line list, object, or parameter list
//     and removed when the construct stays on one line
//
// Indentation is two spaces per open brace. Comments are kept.
func Format(file *File) string {
	type piece struct {
		start int
		text  string
		line  int
		kind  TriviaKind
	}
	pieces := make([]piece, 0, len(file.Tokens)+len(file.Trivia))
	for _, t := range file.Trivia {
		text := ""
		if t.Kind == Comment {
			text = strings.TrimSpace(t.Text)
		}
		pieces = append(pieces, piece{start: t.Start, text: text, line: t.Range.Start.Line, kind: t.Kind})
	}
	for _, token := range file.Tokens {
		if token.Kind == EOF {
			continue
		}
		// A semicolon is a reader convenience, not part of the canonical form.
		if token.Kind == Punct && token.Text == ";" {
			continue
		}
		pieces = append(pieces, piece{start: token.Start, text: rawToken(token), line: token.Line})
	}
	sort.SliceStable(pieces, func(i, j int) bool { return pieces[i].start < pieces[j].start })
	breaks := semicolonBreaks(file.Tokens)

	var out strings.Builder
	depth := 0
	currentLine := 0
	lineDepth := 0
	var lineTokens []string
	pendingCloser := false
	// commaSeparated records, for each open bracket, whether its entries
	// separate with commas. A list and a parameter list always do, a record does
	// too, and a block body ends its entries at a line break instead, so only a
	// block's last entry goes without a trailing comma.
	var commaSeparated []bool
	lastToken := ""
	flush := func() {
		if pendingCloser && len(lineTokens) != 0 {
			last := lineTokens[len(lineTokens)-1]
			if len(commaSeparated) != 0 && commaSeparated[len(commaSeparated)-1] && !isCloser(last) && last != "," && last != "{" && last != "[" && last != "(" && !strings.HasPrefix(last, "//") && !strings.HasPrefix(last, "/*") {
				lineTokens = append(lineTokens, ",")
			}
		}
		pendingCloser = false
		if len(lineTokens) == 0 {
			out.WriteString("\n")
			return
		}
		indent := lineDepth
		if indent < 0 {
			indent = 0
		}
		out.WriteString(strings.Repeat("  ", indent))
		out.WriteString(joinLine(lineTokens))
		out.WriteString("\n")
		lineTokens = lineTokens[:0]
	}
	for index, p := range pieces {
		if p.kind == Newline {
			// A newline before a closer means the construct spans lines.
			for next := index + 1; next < len(pieces); next++ {
				if pieces[next].kind == Newline || pieces[next].text == "" {
					continue
				}
				pendingCloser = isCloser(pieces[next].text)
				break
			}
			flush()
			continue
		}
		if p.text == "" {
			continue
		}
		if (p.line != currentLine || breaks[p.start]) && len(lineTokens) != 0 {
			pendingCloser = isCloser(p.text)
			flush()
		}
		if len(lineTokens) == 0 {
			lineDepth = depth
			if p.text == "}" || p.text == ")" || p.text == "]" {
				lineDepth--
			}
		}
		currentLine = p.line
		lineTokens = append(lineTokens, p.text)
		switch p.text {
		case "{":
			// A brace after a colon, a bracket, a comma, an equals sign, or an arrow
			// opens a record; after a name it opens a block.
			commaSeparated = append(commaSeparated, lastToken == ":" || lastToken == "(" || lastToken == "[" || lastToken == "," || lastToken == "=" || lastToken == ">")
			depth++
		case "(", "[":
			commaSeparated = append(commaSeparated, true)
			depth++
		case "}", ")", "]":
			if len(commaSeparated) != 0 {
				commaSeparated = commaSeparated[:len(commaSeparated)-1]
			}
			if depth > 0 {
				depth--
			}
		}
		lastToken = p.text
	}
	if len(lineTokens) != 0 {
		flush()
	}
	return out.String()
}

// FormatChecked formats a file only when the result can replace the source.
// A file with diagnostics may have lost bytes in the lexer (an unclosed string,
// an unexpected character), so formatting it would drop what the user typed.
// The output must parse cleanly too, so a formatter fault never reaches a file
// or an editor buffer.
func FormatChecked(file *File) (string, bool) {
	if len(file.Diagnostics) != 0 {
		return "", false
	}
	formatted := Format(file)
	if len(Parse(file.Path, formatted).Diagnostics) != 0 {
		return "", false
	}
	return formatted, true
}

// semicolonBreaks returns the token offsets that must start a new line because
// a semicolon separated two entries on one line. Dropping the semicolon alone
// would join the entries into one the parser rejects. As gofmt does, the entry
// after the semicolon goes on its own line, and so do the first entry and the
// closing brace of every enclosing block that shared the line.
func semicolonBreaks(tokens []Token) map[int]bool {
	breaks := map[int]bool{}
	type open struct {
		index int
		split bool
	}
	var stack []open
	for i, token := range tokens {
		if token.Kind != Punct {
			continue
		}
		switch token.Text {
		case "{", "(", "[":
			stack = append(stack, open{index: i})
		case "}", ")", "]":
			if len(stack) == 0 {
				continue
			}
			if stack[len(stack)-1].split {
				breaks[token.Start] = true
			}
			stack = stack[:len(stack)-1]
		case ";":
			next := i + 1
			for next < len(tokens) && tokens[next].Kind == Punct && tokens[next].Text == ";" {
				next++
			}
			if next >= len(tokens) || tokens[next].Kind == EOF || tokens[next].Line != token.Line || tokens[next].Kind == Punct && isCloser(tokens[next].Text) {
				continue
			}
			breaks[tokens[next].Start] = true
			for depth := len(stack) - 1; depth >= 0 && !stack[depth].split && tokens[stack[depth].index].Text == "{"; depth-- {
				stack[depth].split = true
				breaks[tokens[stack[depth].index+1].Start] = true
			}
		}
	}
	return breaks
}

// joinLine applies spacing between the tokens of one line, then removes a
// trailing comma, which only belongs on a construct that spans lines.
func joinLine(tokens []string) string {
	// A trailing comma belongs on a construct that spans lines. When the
	// closing bracket is on the same line, the comma is removed.
	if len(tokens) >= 2 && isCloser(tokens[len(tokens)-1]) && tokens[len(tokens)-2] == "," {
		tokens = append(tokens[:len(tokens)-2], tokens[len(tokens)-1])
	}
	var out strings.Builder
	var previous string
	for index, token := range tokens {
		if index > 0 && needsSpace(previous, token) {
			out.WriteString(" ")
		}
		out.WriteString(token)
		previous = token
	}
	return out.String()
}

func needsSpace(previous, current string) bool {
	// "->" is two tokens and reads as one operator.
	if previous == "-" || previous == "->" {
		return false
	}
	if previous == ":" {
		return true
	}
	if previous == "(" || previous == "[" || previous == "<" || previous == "." {
		return false
	}
	if current == ")" || current == "]" || current == ">" || current == "," || current == "?" || current == "." || current == ":" {
		return false
	}
	// A call or a generic application attaches to its name.
	if current == "(" || current == "[" || current == "<" {
		return false
	}
	// A closing brace that stays on a line takes a space before it, unless the
	// record is empty.
	if current == "}" {
		return previous != "{"
	}
	return true
}

func isCloser(token string) bool {
	return token == "]" || token == "}" || token == ")"
}
