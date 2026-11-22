package chiklang

import (
	"strings"
	"testing"
)

// TestRoundtripKeepsCommentsAndSpacing proves the tree keeps trivia, which is
// the property the roundtrip corpus exists to protect.
func TestRoundtripKeepsCommentsAndSpacing(t *testing.T) {
	source := `// Project services.
services {
  posts {
    operations {
      // Reads one post.
      get(id: id) -> { post: { id: id } }; /* trailing */
    }
  }
}
`
	file := Parse("chik/api.chik", source)
	if got := Serialize(file); got != source {
		t.Fatalf("roundtrip changed the source\nwant:\n%s\ngot:\n%s", source, got)
	}
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
}

// TestUnknownTypeReportsTheReference keeps the diagnostic on the reference and
// not on the declaration, which is what a reader needs to fix the line.
func TestUnknownTypeReportsTheReference(t *testing.T) {
	file := Parse("chik/api.chik", "types {\n  Post = { author: Author };\n}\n")
	// A types block is outside the beta, so CHK0601 comes first. The semantic
	diagnostic := findDiagnostic(file, CodeUnknownType)
	if diagnostic == nil {
		t.Fatalf("no unknown-type diagnostic: %+v", file.Diagnostics)
	}
	got := *diagnostic
	if got.Code != CodeUnknownType || got.Path != "types.Post.author" {
		t.Fatalf("diagnostic = %s %q", got.Code, got.Path)
	}
	if got.Range.Start.Line != 2 || got.Range.Start.Column != 20 {
		t.Fatalf("range starts at %+v, want line 2 column 20", got.Range.Start)
	}
}

// TestCyclicTypeIsRejected covers the self-reference rule.
func TestCyclicTypeIsRejected(t *testing.T) {
	file := Parse("chik/api.chik", "types {\n  Node = { next: Node };\n}\n")
	if findDiagnostic(file, CodeCyclicType) == nil {
		t.Fatalf("diagnostics = %+v", file.Diagnostics)
	}
}

// findDiagnostic returns the first diagnostic with a code, or nil. A file may
// carry a scope diagnostic before the semantic one.
func findDiagnostic(file *File, code string) *Diagnostic {
	for index := range file.Diagnostics {
		if file.Diagnostics[index].Code == code {
			return &file.Diagnostics[index]
		}
	}
	return nil
}

// TestStringEscapesRoundtrip covers the case a naive serializer breaks.
func TestStringEscapesRoundtrip(t *testing.T) {
	source := `project {
  name: "a\"b\\c\nd\u00e9";
}
`
	file := Parse("chik/project.chik", source)
	if got := Serialize(file); got != source {
		t.Fatalf("roundtrip changed the source\nwant:\n%s\ngot:\n%s", source, got)
	}
}

// TestEmptyFileIsReported covers the smallest broken input.
func TestEmptyFileIsReported(t *testing.T) {
	file := Parse("chik/api.chik", "// only a comment\n")
	if len(file.Diagnostics) == 0 || file.Diagnostics[0].Code != CodeEmptyFile {
		t.Fatalf("diagnostics = %+v", file.Diagnostics)
	}
	if got := Serialize(file); !strings.Contains(got, "// only a comment") {
		t.Fatalf("serialize dropped the comment: %q", got)
	}
}
