package chiklang

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

// JSONVersion is the schema version of the JSON form. It changes only when the
// shape breaks, so an agent reading it can rely on the field names.
const JSONVersion = 1

// Render returns the human form of every diagnostic in the file. The text is
// built from the same Error type a command returns, so a printed diagnostic and
// a returned one cannot drift apart.
func Render(file *File) string {
	if len(file.Diagnostics) == 0 {
		return ""
	}
	diagnostics := file.Diagnostics
	for index := range diagnostics {
		if diagnostics[index].File == "" {
			diagnostics[index].File = file.Path
		}
	}
	rendered := (&Error{Diagnostics: diagnostics, Sources: map[string]string{file.Path: file.Source}}).Error()
	return rendered + "\n"
}

// sourceLines splits the source into lines without their line ending. The
// grammar accepts CRLF, so a trailing CR is part of the ending and would
// otherwise print as a stray control character at the end of every quoted
// line.
func sourceLines(source string) []string {
	lines := strings.Split(source, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

func renderDiagnostic(out *strings.Builder, path string, lines []string, diagnostic Diagnostic) {
	severity := diagnostic.Severity
	if severity == "" {
		severity = "error"
	}
	message := diagnostic.Message
	if message == "" {
		message = summaryOf(diagnostic.Code)
	}
	fmt.Fprintf(out, "%s[%s]: %s\n", severity, diagnostic.Code, message)
	start, end := diagnostic.Range.Start, diagnostic.Range.End
	fmt.Fprintf(out, "  --> %s:%d:%d\n", path, start.Line, start.Column)

	// The gutter is as wide as the largest line number, so every source line
	// lines up under its number.
	gutter := len(strconv.Itoa(max(start.Line, end.Line)))
	pad := strings.Repeat(" ", gutter+1)
	sourceLine := lineAt(lines, start.Line)
	fmt.Fprintf(out, "%s |\n", pad)
	display := expandTabs(sourceLine)
	fmt.Fprintf(out, " %*d | %s\n", gutter, start.Line, display)
	out.WriteString(pad)
	out.WriteString(" | ")
	out.WriteString(strings.Repeat(" ", displayColumn(sourceLine, start.Column)-1))
	out.WriteString(strings.Repeat("^", displayWidth(sourceLine, diagnostic.Range)))
	out.WriteString("\n")
	if end.Line > start.Line {
		// A range across lines underlines only its first line, then shows the
		// remaining lines without an underline.
		for line := start.Line + 1; line <= end.Line && line <= len(lines); line++ {
			fmt.Fprintf(out, " %*d | %s\n", gutter, line, expandTabs(lineAt(lines, line)))
		}
	}
	fmt.Fprintf(out, "%s |\n", pad)
	if diagnostic.Help != "" {
		fmt.Fprintf(out, "%s = help: %s\n", pad, diagnostic.Help)
	} else if explanation, ok := LookupExplanation(diagnostic.Code); ok {
		fmt.Fprintf(out, "%s = help: %s\n", pad, explanation.Help)
	}
	notes := diagnostic.Notes
	if len(notes) == 0 {
		if explanation, ok := LookupExplanation(diagnostic.Code); ok {
			notes = explanation.Notes
		}
	}
	for _, note := range notes {
		fmt.Fprintf(out, "%s = note: %s\n", pad, note)
	}
	if diagnostic.Suggestion != nil {
		fmt.Fprintf(out, "%s = suggested: %s\n", pad, diagnostic.Suggestion.Text)
	}
}

// tabWidth is the column a tab advances to, matching the Rust compiler's
// rendering. The grammar allows a tab as whitespace, so a line can contain one.
const tabWidth = 4

// expandTabs replaces each tab with the spaces it occupies. Printing the tab
// itself would let the terminal choose a different width and move the carets
// away from the text.
func expandTabs(line string) string {
	var out strings.Builder
	column := 0
	for _, r := range line {
		if r == '\t' {
			padding := tabWidth - column%tabWidth
			out.WriteString(strings.Repeat(" ", padding))
			column += padding
			continue
		}
		out.WriteRune(r)
		column += runeWidth(r)
	}
	return out.String()
}

// displayColumn is the one-based display column of a source column. Tabs and
// wide characters before it take more than one cell.
func displayColumn(line string, column int) int {
	target := max(1, column)
	display := 1
	index := 1
	for _, r := range line {
		if index >= target {
			return display
		}
		if r == '\t' {
			display += tabWidth - (display-1)%tabWidth
		} else {
			display += runeWidth(r)
		}
		index++
	}
	return display + max(0, target-index)
}

// displayWidth is the number of cells the range covers on its first line. A
// range that spans lines underlines to the end of the first line, and a range
// that ends before it starts still marks one column, so the caret is never
// invisible.
func displayWidth(line string, span Range) int {
	start := displayColumn(line, span.Start.Column)
	end := displayColumn(line, span.End.Column)
	if span.End.Line != span.Start.Line || end <= start {
		end = displayColumn(line, len([]rune(line))+1)
	}
	return max(1, end-start)
}

// runeWidth is the number of terminal cells a character occupies.
func runeWidth(r rune) int {
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	switch width.LookupRune(r).Kind() {
	case width.EastAsianWide, width.EastAsianFullwidth:
		return 2
	}
	return 1
}

func lineAt(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}
	index := line - 1
	return lines[index]
}

func summaryOf(code string) string {
	if explanation, ok := LookupExplanation(code); ok {
		return explanation.Summary
	}
	return "declaration error"
}

type jsonReport = DiagnosticReport

// DiagnosticReport is the JSON form of one file's diagnostics, in the shape
// spec/error-presentation.md fixes. JSON marshals it; a caller that needs the
// fields rather than bytes, such as the language server, reads the struct.
type DiagnosticReport struct {
	Version     int                `json:"version"`
	File        string             `json:"file"`
	Diagnostics []ReportDiagnostic `json:"diagnostics"`
}

// ReportDiagnostic is one entry of DiagnosticReport.
type ReportDiagnostic struct {
	Code       string      `json:"code"`
	Severity   string      `json:"severity"`
	Message    string      `json:"message"`
	Range      Range       `json:"range"`
	Path       string      `json:"path"`
	SourceLine string      `json:"source_line"`
	Help       *string     `json:"help"`
	Notes      []string    `json:"notes"`
	Suggestion *Suggestion `json:"suggestion"`
}

type jsonDiagnostic = ReportDiagnostic

// Report returns the same content JSON marshals, as a struct. The language
// server needs the fields, and building them twice would let the two drift.
func Report(file *File) DiagnosticReport {
	lines := sourceLines(file.Source)
	report := DiagnosticReport{Version: JSONVersion, File: file.Path, Diagnostics: []ReportDiagnostic{}}
	for _, diagnostic := range ordered(file.Diagnostics) {
		severity := diagnostic.Severity
		if severity == "" {
			severity = "error"
		}
		message := diagnostic.Message
		if message == "" {
			message = summaryOf(diagnostic.Code)
		}
		notes := diagnostic.Notes
		if len(notes) == 0 {
			if explanation, ok := LookupExplanation(diagnostic.Code); ok {
				notes = explanation.Notes
			}
		}
		if notes == nil {
			notes = []string{}
		}
		entry := ReportDiagnostic{
			Code: diagnostic.Code, Severity: severity, Message: message,
			Range: diagnostic.Range, Path: diagnostic.Path,
			SourceLine: lineAt(lines, diagnostic.Range.Start.Line),
			Notes:      notes, Suggestion: diagnostic.Suggestion,
		}
		if diagnostic.Help != "" {
			help := diagnostic.Help
			entry.Help = &help
		} else if explanation, ok := LookupExplanation(diagnostic.Code); ok {
			help := explanation.Help
			entry.Help = &help
		}
		report.Diagnostics = append(report.Diagnostics, entry)
	}
	return report
}

// JSON returns the same content as the human form in the shape spec/error-presentation.md
// fixes. The order is start position, then code, so two implementations agree.
// The object is one line, so a caller reporting several files can print one
// object per line and a reader can decode them one at a time.
func JSON(file *File) ([]byte, error) {
	data, err := json.Marshal(Report(file))
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ordered returns the diagnostics by start position, then by code, which is
// the order both output forms promise.
func ordered(diagnostics []Diagnostic) []Diagnostic {
	result := append([]Diagnostic(nil), diagnostics...)
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i].Range.Start, result[j].Range.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return result[i].Code < result[j].Code
	})
	return result
}
