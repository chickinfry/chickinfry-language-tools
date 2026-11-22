package chiklang

import (
	"strings"
	"testing"
)

const editAPI = `// header comment
services {
  posts {
    operations {
      get(id: id) -> { title: string }
      list() -> { title: string }
    }
  }
}
`

// TestEditReplacesOneUnitKeepsComments proves a unit rewrite leaves the rest of
// the file, including comments attached to other units, untouched.
func TestEditReplacesOneUnitKeepsComments(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	result := Edit(file, []Change{{Path: "services.posts.operations.list", Text: "list() -> { title: string, body: string }"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	for _, want := range []string{"// header comment", "get(id: id) -> { title: string }", "list() -> { title: string, body: string }"} {
		if !strings.Contains(result.Source, want) {
			t.Fatalf("output lacks %q:\n%s", want, result.Source)
		}
	}
}

// TestEditIsIdempotent proves a rewritten file is already in the formatter's
// canonical layout, which is the .chik roundtrip contract.
func TestEditIsIdempotent(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	result := Edit(file, []Change{{Path: "services.posts.operations.get", Text: "get(id: id,  slug: string) -> { title: string }"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if again := Edit(Parse("chik/api.chik", result.Source), nil); again.Source != result.Source {
		t.Fatalf("format is not idempotent:\n%s\n---\n%s", result.Source, again.Source)
	}
}

// TestEditDeletesAUnit proves a delete removes the unit and its line without
// leaving a blank line behind.
func TestEditDeletesAUnit(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	result := Edit(file, []Change{{Path: "services.posts.operations.list"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if strings.Contains(result.Source, "list()") {
		t.Fatalf("unit survived:\n%s", result.Source)
	}
	if strings.Contains(result.Source, "\n\n\n") {
		t.Fatalf("delete left a blank line:\n%s", result.Source)
	}
}

// TestEditRejectsUnknownAndOverlappingPaths keeps a caller from silently
// editing nothing or claiming the same bytes twice.
func TestEditRejectsUnknownAndOverlappingPaths(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	if result := Edit(file, []Change{{Path: "services.nope", Text: "x"}}); len(result.Diagnostics) == 0 {
		t.Fatal("unknown path accepted")
	}
	overlap := []Change{
		{Path: "services.posts", Text: "posts {}"},
		{Path: "services.posts.operations.get", Text: "get() -> {}"},
	}
	if result := Edit(file, overlap); len(result.Diagnostics) == 0 {
		t.Fatal("overlapping paths accepted")
	}
}

// TestEditRefusesBrokenSource keeps a syntax error from being rewritten into a
// different file.
func TestEditRefusesBrokenSource(t *testing.T) {
	file := Parse("chik/api.chik", "services {\n  posts {\n")
	if len(file.Diagnostics) == 0 {
		t.Fatal("expected a parse diagnostic")
	}
	result := Edit(file, []Change{{Path: "services.posts", Text: "posts {}"}})
	if len(result.Diagnostics) == 0 || result.Source != file.Source {
		t.Fatalf("broken source was edited: %+v", result)
	}
}

// TestEditRefusesACandidateThatDoesNotParse proves the result is re-parsed, so
// a change that produces invalid source is not published.
func TestEditRefusesACandidateThatDoesNotParse(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	result := Edit(file, []Change{{Path: "services.posts.operations.get", Text: "get(id: id -> {"}})
	if len(result.Diagnostics) == 0 {
		t.Fatal("invalid candidate accepted")
	}
	if result.Source != file.Source {
		t.Fatal("a failed edit returned a candidate")
	}
}

// TestEditAddsANewOperation proves a path that does not exist yet is an
// addition. Without it the console could only replace operations, never create
// one, and a new unit reported "no unit at this path".
func TestEditAddsANewOperation(t *testing.T) {
	file := Parse("chik/api.chik", editAPI)
	result := Edit(file, []Change{{Path: "services.posts.operations.list", Text: "list() -> { title: string }"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if !strings.Contains(result.Source, "list() -> { title: string }") {
		t.Fatalf("operation missing:\n%s", result.Source)
	}
	if !strings.Contains(result.Source, "// header comment") {
		t.Fatalf("comment lost:\n%s", result.Source)
	}
	// The result must be canonical, or the next fmt run would rewrite it again.
	if again := Edit(Parse("chik/api.chik", result.Source), nil); again.Source != result.Source {
		t.Fatalf("added unit is not canonical:\n%s\n---\n%s", result.Source, again.Source)
	}
	if strings.Contains(result.Source, "\n\n\n") {
		t.Fatalf("addition left a blank line:\n%s", result.Source)
	}
}
