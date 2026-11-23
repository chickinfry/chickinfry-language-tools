package chiklang

import (
	"encoding/json"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TestRenderMatchesPublishedShape pins the human form to the layout
// spec/error-presentation.md shows, because a reader learns the shape once and
// then reads every error the same way.
func TestRenderMatchesPublishedShape(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      get(id: id) -> { author: Author };\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	want := "error[CHK0301]: unknown type\n" +
		"  --> chik/api.chik:4:32\n" +
		"   |\n" +
		" 4 |       get(id: id) -> { author: Author };\n" +
		"   | " + strings.Repeat(" ", 31) + "^^^^^^\n" +
		"   |\n" +
		"   = help: declare the type in types { } or use a built-in\n" +
		"   = note: built-in types are id, string, and timestamp\n"
	if got := Render(file); got != want {
		t.Fatalf("render =\n%s\nwant:\n%s", got, want)
	}
}

// TestRenderCountsColumnsInCodePoints keeps a non-ASCII line aligned. A byte
// count would put the caret under the wrong character.
func TestRenderCountsColumnsInCodePoints(t *testing.T) {
	source := "types {\n  Post = { note: string, author: Author };\n}\n"
	file := Parse("chik/api.chik", source)
	rendered := Render(file)
	if !strings.Contains(rendered, "\n   | "+strings.Repeat(" ", 33)+"^^^^^^\n") {
		t.Fatalf("caret is not under Author:\n%s", rendered)
	}
}

// TestJSONMatchesPublishedFields checks the field names an agent reads. The
// schema fixes them, so a rename breaks the agent gate.
func TestJSONMatchesPublishedFields(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      get(id: id) -> { author: Author };\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	data, err := JSON(file)
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Version     int    `json:"version"`
		File        string `json:"file"`
		Diagnostics []struct {
			Code       string   `json:"code"`
			Severity   string   `json:"severity"`
			Message    string   `json:"message"`
			Path       string   `json:"path"`
			SourceLine string   `json:"source_line"`
			Help       *string  `json:"help"`
			Notes      []string `json:"notes"`
			Range      struct {
				Start Pos `json:"start"`
				End   Pos `json:"end"`
			} `json:"range"`
			Suggestion *Suggestion `json:"suggestion"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != JSONVersion || report.File != "chik/api.chik" || len(report.Diagnostics) != 1 {
		t.Fatalf("report = %s", data)
	}
	got := report.Diagnostics[0]
	if got.Code != CodeUnknownType || got.Severity != "error" {
		t.Fatalf("diagnostic = %+v", got)
	}
	if got.Range.Start.Line != 4 || got.Range.Start.Column != 32 || got.Range.End.Column != 38 {
		t.Fatalf("range = %+v", got.Range)
	}
	if got.SourceLine != "      get(id: id) -> { author: Author };" || got.Help == nil || len(got.Notes) != 1 || got.Suggestion != nil {
		t.Fatalf("detail = %+v", got)
	}
}

// TestDiagnosticsAreOrdered checks the promised order: start position, then
// code. Two implementations must agree even when the parser finds them in a
// different sequence.
func TestDiagnosticsAreOrdered(t *testing.T) {
	file := &File{Path: "chik/api.chik", Source: "one\ntwo\n", Diagnostics: []Diagnostic{
		{Code: CodeExpectedName, Range: Range{Start: Pos{Line: 2, Column: 1}, End: Pos{Line: 2, Column: 2}}},
		{Code: CodeExpectedToken, Range: Range{Start: Pos{Line: 1, Column: 3}, End: Pos{Line: 1, Column: 4}}},
		{Code: CodeEmptyFile, Range: Range{Start: Pos{Line: 1, Column: 3}, End: Pos{Line: 1, Column: 4}}},
	}}
	ordered := ordered(file.Diagnostics)
	want := []string{CodeExpectedToken, CodeEmptyFile, CodeExpectedName}
	for index, code := range want {
		if ordered[index].Code != code {
			t.Fatalf("order = %v", ordered)
		}
	}
	if file.Diagnostics[0].Code != CodeExpectedName {
		t.Fatal("ordering mutated the file's diagnostics")
	}
}

// TestRepeatedDiagnosticsCollapse keeps a recovery cascade from printing the
// same problem several times. The parser can reach the same token through more
// than one recovery path, and repeating it buries the problem that caused it.
func TestRepeatedDiagnosticsCollapse(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      get(name: \"abc) -> {}\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	seen := map[string]bool{}
	for _, diagnostic := range file.Diagnostics {
		key := diagnostic.Code + "|" + fmt.Sprint(diagnostic.Range)
		if seen[key] {
			t.Fatalf("the same diagnostic appears twice: %s %+v", diagnostic.Code, diagnostic.Range)
		}
		seen[key] = true
	}
}

// TestRenderUnderlinesMultiLineRanges checks the published rule: a range that
// spans lines underlines to the end of the first line and shows the rest
// without carets.
func TestRenderUnderlinesMultiLineRanges(t *testing.T) {
	file := &File{Path: "chik/api.chik", Source: "one two\nthree\n", Diagnostics: []Diagnostic{{
		Code: CodeExpectedName, Message: "expected a name",
		Range: Range{Start: Pos{Line: 1, Column: 5}, End: Pos{Line: 2, Column: 3}},
	}}}
	want := "error[CHK0103]: expected a name\n" +
		"  --> chik/api.chik:1:5\n" +
		"   |\n" +
		" 1 | one two\n" +
		"   |     ^^^\n" +
		" 2 | three\n" +
		"   |\n" +
		"   = help: write an identifier\n"
	if got := Render(file); got != want {
		t.Fatalf("render =\n%s\nwant:\n%s", got, want)
	}
}

// TestStringTokenRangeUsesSourceWidth keeps a diagnostic after a string on the
// same line at the right column. The decoded text is shorter than the spelling
// once an escape appears, so a range built from it would drift left.
func TestStringTokenRangeUsesSourceWidth(t *testing.T) {
	source := "project {\n  name: \"caf\\u00e9\" server: 3;\n}\n"
	file := Parse("chik/project.chik", source)
	if len(file.Diagnostics) == 0 {
		t.Fatal("the unquoted value was accepted")
	}
	// "caf\u00e9" is 9 code points in the source but 4 decoded, so a range
	// built from the decoded text would report column 13.
	first := ordered(file.Diagnostics)[0]
	if first.Range.Start.Line != 2 || first.Range.Start.Column != 20 {
		t.Fatalf("diagnostic starts at %+v, want 2:20", first.Range.Start)
	}
}

// TestCRLFSourceHasNoStrayCarriageReturn keeps the rendered line and the JSON
// source line free of the line ending. The grammar accepts CRLF, and a CR left
// in the quoted line moves the terminal cursor to column one.
func TestCRLFSourceHasNoStrayCarriageReturn(t *testing.T) {
	// Two entries on one line need a separator, which is the diagnostic this
	// case needs to render a source line at all.
	source := "project {\r\n  name: \"demo\" server: \"typescript\"\r\n}\r\n"
	file := Parse("chik/project.chik", source)
	if len(file.Diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(file.Diagnostics), file.Diagnostics)
	}
	rendered := Render(file)
	if strings.Contains(rendered, "\r") {
		t.Fatalf("rendered form keeps a carriage return: %q", rendered)
	}
	if !strings.Contains(rendered, " 2 |   name: \"demo\" server: \"typescript\"\n") {
		t.Fatalf("source line is not clean:\n%q", rendered)
	}
	data, err := JSON(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "\\r") {
		t.Fatalf("JSON source line keeps a carriage return: %s", data)
	}
}

// TestUnderlineLandsOnTheReportedColumn checks the caret by display cell, not
// by code point. A tab and a wide character each take more than one cell, so a
// caret placed by counting runes would point at the wrong text.
func TestUnderlineLandsOnTheReportedColumn(t *testing.T) {
	for _, test := range []struct {
		name, line string
		want       string
	}{
		{"tab", "\tname: \"demo\" server: 1;", "server"},
		{"wide", "  name: \"\ud55c\uad6d\uc5b4\" server: 1;", "server"},
		{"ascii", "  name: \"demo\" server: 1;", "server"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// The column is the word's code point position in the source, which
			// is what a diagnostic reports.
			byteIndex := strings.Index(test.line, test.want)
			column := len([]rune(test.line[:byteIndex])) + 1
			file := &File{Path: "chik/project.chik", Source: test.line + "\n", Diagnostics: []Diagnostic{{
				Code: CodeExpectedToken, Message: "expected \";\"",
				Range: Range{Start: Pos{Line: 1, Column: column}, End: Pos{Line: 1, Column: column + 1}},
			}}}
			rendered := Render(file)
			lines := strings.Split(rendered, "\n")
			var sourceRow, caretRow string
			for index, line := range lines {
				if strings.HasPrefix(line, " 1 | ") {
					sourceRow, caretRow = line, lines[index+1]
				}
			}
			if sourceRow == "" {
				t.Fatalf("no source row:\n%s", rendered)
			}
			// A raw tab would be laid out by the terminal at its own width,
			// which is what the expansion exists to avoid.
			if strings.ContainsRune(sourceRow, '\t') {
				t.Fatalf("source row keeps a raw tab: %q", sourceRow)
			}
			offset := strings.Index(caretRow, "^")
			if offset < 0 {
				t.Fatalf("no caret:\n%s", rendered)
			}
			// Both rows start with the gutter, which is not part of the source.
			prefix := strings.Index(sourceRow, "| ") + 2
			offset -= prefix
			// The caret sits on a display cell, and a wide character takes two
			// cells, so the cell is mapped back to the rune it covers.
			index := runeAtCell(sourceRow[prefix:], offset)
			if got := string([]rune(sourceRow[prefix:])[index:]); !strings.HasPrefix(got, test.want) {
				t.Fatalf("caret points at %q, want %q\n%s", got, test.want, rendered)
			}
		})
	}
}

// runeAtCell returns the index of the rune that starts at a zero-based display
// cell.
func runeAtCell(line string, cell int) int {
	position := 0
	for index, r := range []rune(line) {
		if position >= cell {
			return index
		}
		if r == '\t' {
			position += tabWidth - position%tabWidth
			continue
		}
		position += runeWidth(r)
	}
	return len([]rune(line))
}

// TestEveryPublishedNumberHasAnExplanation catches a number added to the
// constants without a matching entry, which would leave chik explain empty for
// a code users can see. The numbers are read from the source so a new constant
// cannot pass by being absent from a hand-kept list.
func TestEveryPublishedNumberHasAnExplanation(t *testing.T) {
	codes := declaredCodes(t)
	if len(codes) == 0 {
		t.Fatal("no CHK constants found")
	}
	for _, code := range codes {
		explanation, ok := LookupExplanation(code)
		if !ok {
			t.Errorf("%s has no explanation", code)
			continue
		}
		if explanation.Summary == "" || explanation.Help == "" || explanation.Body == "" {
			t.Errorf("%s has an incomplete explanation", code)
		}
	}
	if len(Explanations()) != len(codes) {
		t.Fatalf("%d explanations for %d numbers", len(Explanations()), len(codes))
	}
}

func declaredCodes(t *testing.T) []string {
	t.Helper()
	parsed, err := goparser.ParseFile(token.NewFileSet(), "token.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, declaration := range parsed.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Values) != 1 {
				continue
			}
			literal, ok := value.Values[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				continue
			}
			code, err := strconv.Unquote(literal.Value)
			if err == nil && strings.HasPrefix(code, "CHK") {
				codes = append(codes, code)
			}
		}
	}
	return codes
}
