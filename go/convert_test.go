package chiklang

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConvertCasesMatchCorpus compares the converter with the corpus. The role
// comes from the expected file name, so a case states which declaration it is.
func TestConvertCasesMatchCorpus(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "convert", "*", "input.yml"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no convert cases: %v", err)
	}
	for _, path := range cases {
		dir := filepath.Dir(path)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := filepath.Glob(filepath.Join(dir, "expected", "*.chik"))
			if err != nil || len(expected) != 1 {
				t.Fatalf("expected exactly one .chik file, found %v", expected)
			}
			role := roleOf(filepath.Base(expected[0]))
			want, err := os.ReadFile(expected[0])
			if err != nil {
				t.Fatal(err)
			}
			got, diagnostics := Convert(role, input)
			if len(diagnostics) != 0 {
				t.Fatalf("conversion refused: %+v", diagnostics)
			}
			if got != string(want) {
				t.Fatalf("conversion differs\nwant:\n%s\ngot:\n%s", want, got)
			}
			// The result must parse, or the converter produced something the
			// language does not accept.
			if parsed := Parse("chik/"+filepath.Base(expected[0]), got); len(parsed.Diagnostics) != 0 {
				t.Fatalf("converted output does not parse: %+v", parsed.Diagnostics)
			}
		})
	}
}

func roleOf(name string) string {
	return name[:len(name)-len(".chik")]
}

// TestConvertedOutputIsCanonical keeps conversion and formatting from
// disagreeing: a converted file must already be what chik fmt would produce,
// or the first fmt after a conversion would change it.
func TestConvertedOutputIsCanonical(t *testing.T) {
	source := "project:\n  name: demo\ngeneration:\n  server:\n    language: typescript\n  clients: [typescript]\n"
	converted, diagnostics := Convert("project", []byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("refused: %+v", diagnostics)
	}
	if formatted := Format(Parse("chik/project.chik", converted)); formatted != converted {
		t.Fatalf("conversion is not canonical\nconverted:\n%s\nformatted:\n%s", converted, formatted)
	}
}
