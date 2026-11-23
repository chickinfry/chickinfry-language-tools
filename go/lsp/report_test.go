package lsp

import (
	"encoding/json"
	"testing"

	"github.com/chickinfry/chik-lang/go"
)

// The instruction for this server is that diagnostics carry the same content
// chik check --format json prints. This test holds the two to one source: the
// protocol fields are filled from chiklang.Report, so a code, path, help line,
// or note that changes there changes here.
func TestPublishedDiagnosticsCarryTheJSONForm(t *testing.T) {
	source := "types {\n  Post = { author: Author }\n}\n"
	uri := "file:///project/chik/api.chik"

	s := &server{documents: map[string]*document{}}
	s.documents[uri] = newDocument(uri, 1, source)

	// What the JSON form says.
	file := chiklang.Parse(pathOf(uri), source)
	report := chiklang.Report(file)
	if len(report.Diagnostics) == 0 {
		t.Fatal("the fixture should report an unknown type")
	}
	want := report.Diagnostics[0]

	// What the server publishes.
	var published struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	payload, err := json.Marshal(map[string]any{"diagnostics": s.diagnosticsFor(uri)})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(payload, &published); err != nil {
		t.Fatal(err)
	}
	if len(published.Diagnostics) != 1 {
		t.Fatalf("published %d diagnostics, want 1", len(published.Diagnostics))
	}
	got := published.Diagnostics[0]

	if got.Code != want.Code {
		t.Errorf("code = %q, JSON form has %q", got.Code, want.Code)
	}
	if got.Path != want.Path {
		t.Errorf("path = %q, JSON form has %q", got.Path, want.Path)
	}
	if got.Message != want.Message {
		t.Errorf("message = %q, JSON form has %q", got.Message, want.Message)
	}
	if got.SourceLine != want.SourceLine {
		t.Errorf("source line = %q, JSON form has %q", got.SourceLine, want.SourceLine)
	}
	if want.Help != nil && (got.Help == nil || *got.Help != *want.Help) {
		t.Errorf("help = %v, JSON form has %q", got.Help, *want.Help)
	}
	if len(got.Notes) != len(want.Notes) {
		t.Errorf("notes = %v, JSON form has %v", got.Notes, want.Notes)
	}
	// The protocol range is zero-based; the JSON form is one-based.
	if got.Range.Start.Line != want.Range.Start.Line-1 {
		t.Errorf("range line = %d, JSON form has %d", got.Range.Start.Line, want.Range.Start.Line)
	}
}
