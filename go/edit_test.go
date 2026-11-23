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

func TestEditDeletesOnlyTheInlineUnit(t *testing.T) {
	for _, tt := range []struct {
		name string
		line string
	}{
		{"before another operation", "/* policy */ list() -> { title: string } /* keep */; count() -> { total: string }"},
		{"after another operation", "count() -> { total: string }; /* policy */ list() -> { title: string } /* keep */"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := strings.Replace(editAPI, "list() -> { title: string }", tt.line, 1)
			result := Edit(Parse("chik/api.chik", source), []Change{{Path: "services.posts.operations.list"}})
			if len(result.Diagnostics) != 0 {
				t.Fatalf("delete: %+v", result.Diagnostics)
			}
			if strings.Contains(result.Source, "list()") || !strings.Contains(result.Source, "count() -> { total: string }") ||
				!strings.Contains(result.Source, "/* policy */") || !strings.Contains(result.Source, "/* keep */") {
				t.Fatalf("source after deletion:\n%s", result.Source)
			}
		})
	}
}

func TestEditDeletesOnlyTheScalarPolicy(t *testing.T) {
	source := `auth {
  methods: [password]
  signup: "open"; /* Keep the session policy. */ session: { idle: "1h" }
}
`
	file := Parse("chik/auth.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("fixture: %+v", file.Diagnostics)
	}
	result := Edit(file, []Change{{Path: "auth.signup"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("delete: %+v", result.Diagnostics)
	}
	for _, want := range []string{"methods: [password]", "/* Keep the session policy. */", "session: {", `idle: "1h"`} {
		if !strings.Contains(result.Source, want) {
			t.Fatalf("source lacks %q:\n%s", want, result.Source)
		}
	}
	if strings.Contains(result.Source, "signup:") {
		t.Fatalf("signup survived:\n%s", result.Source)
	}
}

func TestEditDeletesAnOperationWithARecordInput(t *testing.T) {
	source := strings.Replace(editAPI, "list() -> { title: string }",
		"put(data: { name: string }) -> { ok: string }; count() -> { total: string }", 1)
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("fixture: %+v", file.Diagnostics)
	}
	result := Edit(file, []Change{{Path: "services.posts.operations.put"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("delete: %+v", result.Diagnostics)
	}
	if strings.Contains(result.Source, "put(") || !strings.Contains(result.Source, "count() -> { total: string }") {
		t.Fatalf("source after deletion:\n%s", result.Source)
	}
}

func TestEditDeletesACommaSeparatedColumn(t *testing.T) {
	source := "database { tables { posts { columns { /* keep */ title: { type: string }, id: { type: id } } } } }"
	parsed := Parse("chik/schema.chik", source)
	if len(parsed.Diagnostics) != 0 {
		t.Fatalf("fixture: %+v", parsed.Diagnostics)
	}
	result := Edit(parsed, []Change{{Path: "database.tables.posts.columns.title"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("delete: %+v", result.Diagnostics)
	}
	if strings.Contains(result.Source, "title:") || !strings.Contains(result.Source, "id:") ||
		!strings.Contains(result.Source, "/* keep */") {
		t.Fatalf("source after deletion:\n%s", result.Source)
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
	result := Edit(file, []Change{{Path: "services.posts.operations.summary", Text: "summary() -> { title: string }"}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if !strings.Contains(result.Source, "summary() -> { title: string }") {
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
