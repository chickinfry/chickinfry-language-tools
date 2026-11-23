package chiklang

import (
	"strings"
	"testing"
)

// TestOptionalIsAnInputModifier pins the two spellings of the same rule. A
// parameter may be marked optional in either position, and the word must not
// be read as a field name or as a type.
func TestOptionalIsAnInputModifier(t *testing.T) {
	cases := []struct{ name, source string }{
		{"bare", "services {\n  s {\n    operations {\n      o(a: string optional) -> {}\n    }\n  }\n}\n"},
		{"nullable", "services {\n  s {\n    operations {\n      o(a: string? optional) -> {}\n    }\n  }\n}\n"},
		{"list", "services {\n  s {\n    operations {\n      o(a: { b: string }[] optional) -> {}\n    }\n  }\n}\n"},
		{"nested input", "services {\n  s {\n    operations {\n      o(a: { rows: { title: string optional }[] }) -> {}\n    }\n  }\n}\n"},
		{"named input", "types {\n  Post = { author: { title: string optional } }\n}\nservices {\n  s {\n    operations {\n      o(a: Post) -> {}\n    }\n  }\n}\n"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			file := Parse("chik/api.chik", test.source)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("diagnostics: %+v", file.Diagnostics)
			}
			got, err := ToYAML(file)
			if err != nil {
				t.Fatal(err)
			}
			if want := "optional: true"; !strings.Contains(string(got), want) {
				t.Fatalf("optional lost:\n%s", got)
			}
		})
	}
}

func TestOptionalOnOutputIsRejected(t *testing.T) {
	for _, test := range []struct {
		name, result, path string
	}{
		{"scalar", "{ title: string optional }", "title"},
		{"object", "{ post: { title: string optional } }", "post.title"},
		{"nested list", "{ rows: { post: { title: string optional } }[] }", "rows.post.title"},
	} {
		t.Run(test.name, func(t *testing.T) {
			line := "      o() -> " + test.result
			file := Parse("chik/api.chik", "services {\n  s {\n    operations {\n"+line+"\n    }\n  }\n}\n")
			if len(file.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %+v, want one %s", file.Diagnostics, CodeInvalidModifier)
			}
			diagnostic := file.Diagnostics[0]
			column := strings.Index(line, "optional") + 1
			wantRange := Range{
				Start: Pos{Line: 4, Column: column},
				End:   Pos{Line: 4, Column: column + len("optional")},
			}
			if diagnostic.Code != CodeInvalidModifier || diagnostic.Path != "services.s.o."+test.path || diagnostic.Range != wantRange {
				t.Fatalf("diagnostic = %+v, want %s on optional at %s (%+v)", diagnostic, CodeInvalidModifier, test.path, wantRange)
			}
		})
	}
}

func TestOptionalNamedRecordsAreRejectedAtOutputUse(t *testing.T) {
	source := `types {
  Post = { author: Author }
  Author = { title: string optional }
}
services {
  s {
    operations {
      direct(a: Author) -> { a: Author }
      nested(a: Post) -> { envelope: { rows: Post[] } }
    }
  }
}
`
	file := Parse("chik/api.chik", source)
	want := []struct {
		path string
		at   Range
	}{
		{"services.s.direct.a", Range{Start: Pos{Line: 8, Column: 33}, End: Pos{Line: 8, Column: 39}}},
		{"services.s.nested.envelope.rows", Range{Start: Pos{Line: 9, Column: 46}, End: Pos{Line: 9, Column: 50}}},
	}
	if len(file.Diagnostics) != len(want) {
		t.Fatalf("diagnostics = %+v, want a diagnostic for each output type use", file.Diagnostics)
	}
	for index, diagnostic := range file.Diagnostics {
		expected := want[index]
		if diagnostic.Code != CodeInvalidModifier || diagnostic.Path != expected.path || diagnostic.Range != expected.at {
			t.Errorf("diagnostic = %+v, want %s at %s (%+v)", diagnostic, CodeInvalidModifier, expected.path, expected.at)
		}
	}
}

// TestFieldNamedOptionalStaysAField keeps the word usable as a name. It is not
// reserved, so a field called optional must still be an ordinary field.
func TestFieldNamedOptionalStaysAField(t *testing.T) {
	file := Parse("chik/api.chik", "services {\n  s {\n    operations {\n      o(a: string, optional: string) -> {}\n    }\n  }\n}\n")
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
}

// TestDuplicateCapabilityNameIsRejected covers the name-keyed map rule: the key
// names a capability, so two entries with one key would hide a selection.
func TestDuplicateCapabilityNameIsRejected(t *testing.T) {
	file := Parse("chik/project.chik", "project {\n  name: \"a\"\n  capabilities: { control_plane: \"x\", control_plane: \"y\" }\n}\n")
	if len(file.Diagnostics) != 1 || file.Diagnostics[0].Code != CodeDuplicateName {
		t.Fatalf("diagnostics = %+v, want one %s", file.Diagnostics, CodeDuplicateName)
	}
}
