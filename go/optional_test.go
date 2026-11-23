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
