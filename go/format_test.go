package chiklang

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFormatCasesMatchCorpus compares the formatter output with the corpus byte
// for byte. The corpus decides the layout, not the implementation.
func TestFormatCasesMatchCorpus(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "format", "*", "input.chik"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no format cases: %v", err)
	}
	for _, path := range cases {
		dir := filepath.Dir(path)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join(dir, "expected.chik"))
			if err != nil {
				t.Fatal(err)
			}
			file := Parse(path, string(input))
			got := Format(file)
			if got != string(want) {
				t.Fatalf("format differs\nwant:\n%s\ngot:\n%s", want, got)
			}
			// Formatting its own output must change nothing.
			if again := Format(Parse(path, got)); again != got {
				t.Fatalf("format is not idempotent\nfirst:\n%s\nsecond:\n%s", got, again)
			}
		})
	}
}

// TestFormatKeepsComments proves a comment is never dropped by formatting.
func TestFormatKeepsComments(t *testing.T) {
	source := "// header\nservices {\n  posts {\n    operations {\n      get(id: id) -> { post: { id: id } }; // trailing\n    }\n  }\n}\n"
	got := Format(Parse("chik/api.chik", source))
	for _, want := range []string{"// header", "// trailing"} {
		if !contains(got, want) {
			t.Fatalf("format dropped %q:\n%s", want, got)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// TestFormatDoesNotAddTrailingComma keeps a block's last entry free of a comma.
// The separator decision makes a line break end a block entry, so a comma
// there is not the canonical form; the formatter only keeps or drops one the
// author wrote.
func TestFormatDoesNotAddTrailingComma(t *testing.T) {
	for _, source := range []string{
		"project {\n  name: \"demo\"\n  server: \"typescript\"\n}\n",
		"project {\n     name:   \"demo\"\n  server: \"typescript\"\n}\n",
	} {
		got := Format(Parse("chik/project.chik", source))
		want := "project {\n  name: \"demo\"\n  server: \"typescript\"\n}\n"
		if got != want {
			t.Fatalf("format(%q) = %q, want %q", source, got, want)
		}
	}
}

// TestFormatKeepsCommasInMultilineRecord keeps the separators a multi-line
// record needs. Only a block drops them, because a line break ends its entry.
func TestFormatKeepsCommasInMultilineRecord(t *testing.T) {
	source := "types {\n  Post = {\n    id: id,\n    title: string,\n  }\n}\n"
	got := Format(Parse("chik/api.chik", source))
	if got != source {
		t.Fatalf("format = %q, want %q", got, source)
	}
}

// A semicolon between two entries on one line ends the first entry. Dropping it
// without a line break joined the entries into one the parser rejects, and
// chik fmt wrote that over a valid file.
func TestFormatPutsSemicolonSeparatedEntriesOnTheirOwnLines(t *testing.T) {
	for _, c := range []struct{ name, path, source, want string }{
		{
			name:   "one block",
			path:   "chik/project.chik",
			source: "project { name: \"demo\"; server: \"typescript\" }\n",
			want:   "project {\n  name: \"demo\"\n  server: \"typescript\"\n}\n",
		},
		{
			name:   "nested blocks",
			path:   "chik/api.chik",
			source: "services { posts { operations { get(id: id) -> { id: id }; list() -> { id: id } } } }\n",
			want:   "services {\n  posts {\n    operations {\n      get(id: id) -> { id: id }\n      list() -> { id: id }\n    }\n  }\n}\n",
		},
		{
			name:   "inside a multi-line block",
			path:   "chik/project.chik",
			source: "project {\n  name: \"demo\"; server: \"typescript\"\n}\n",
			want:   "project {\n  name: \"demo\"\n  server: \"typescript\"\n}\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			file := Parse(c.path, c.source)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("the source is valid and should parse: %v", file.Diagnostics)
			}
			got, ok := FormatChecked(file)
			if !ok {
				t.Fatalf("formatted output does not parse:\n%s", Format(file))
			}
			if got != c.want {
				t.Fatalf("format =\n%s\nwant:\n%s", got, c.want)
			}
			if again := Format(Parse(c.path, got)); again != got {
				t.Fatalf("format is not idempotent:\n%s", again)
			}
		})
	}
}

// The lexer drops the bytes of an unclosed string or an unexpected character,
// so the formatter output of such a file lacks them. FormatChecked must refuse
// rather than hand that output to a file or an editor.
func TestFormatCheckedRefusesASourceWithLexerErrors(t *testing.T) {
	for _, source := range []string{
		"project {\n  name: \"dem\n}\n",
		"project {\n  name: \"demo\" # note\n}\n",
		"project {\n  name: '\ub370\ubaa8'\n}\n",
	} {
		file := Parse("chik/project.chik", source)
		if formatted, ok := FormatChecked(file); ok {
			t.Errorf("FormatChecked accepted %q and returned %q", source, formatted)
		}
	}
}
