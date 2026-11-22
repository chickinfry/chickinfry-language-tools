package chiklang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func corpusRoot(t *testing.T) string {
	t.Helper()
	root := ".."
	if _, err := os.Stat(filepath.Join(root, "conformance")); err != nil {
		t.Fatalf("read the conformance corpus: %v", err)
	}
	return root
}

// TestRoundtripCases checks lossless serialization and the formatter fixed point.
func TestRoundtripCases(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "roundtrip", "*", "input.chik"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no roundtrip cases: %v", err)
	}
	for _, path := range cases {
		t.Run(strings.TrimSuffix(filepath.Base(filepath.Dir(path)), ".chik"), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			file := Parse(path, string(source))
			if got := Format(file); got != string(source) {
				t.Fatalf("format changed the corpus fixed point\nwant:\n%s\ngot:\n%s", source, got)
			}
			got := Serialize(file)
			if got != string(source) {
				t.Fatalf("roundtrip changed the source\nwant:\n%s\ngot:\n%s", source, got)
			}
		})
	}
}

// TestFormatCasesParse checks that every format case parses without a syntax
// error, and that the roundtrip of its input is stable.
func TestFormatCasesParse(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "format", "*", "input.chik"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no format cases: %v", err)
	}
	for _, path := range cases {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			file := Parse(path, string(source))
			for _, diagnostic := range file.Diagnostics {
				t.Errorf("%s: %s", diagnostic.Code, diagnostic.Message)
			}
			if got := Serialize(file); got != string(source) {
				t.Fatalf("roundtrip changed the source\nwant:\n%s\ngot:\n%s", source, got)
			}
		})
	}
}
