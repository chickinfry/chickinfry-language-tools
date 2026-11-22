package chiklang

import "strings"

// Error renders diagnostics as the text form a reader expects: a number, a
// position, the source line, an underline, and a help line. It implements
// error so a caller can return it from a command.
type Error struct {
	Diagnostics []Diagnostic
	// Sources maps a file path to its text, so a diagnostic can show the line.
	Sources map[string]string
}

func (e *Error) Error() string {
	var out strings.Builder
	for index, diagnostic := range ordered(e.Diagnostics) {
		if index > 0 {
			out.WriteString("\n")
		}
		source := e.Sources[fileOf(diagnostic)]
		renderDiagnostic(&out, fileOf(diagnostic), sourceLines(source), diagnostic)
	}
	return strings.TrimRight(out.String(), "\n")
}

// fileOf is the file a diagnostic belongs to.
func fileOf(diagnostic Diagnostic) string {
	return diagnostic.File
}
