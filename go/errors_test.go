package chiklang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestErrorCasesMatchCorpus compares the code, path, and range of each
// diagnostic with the corpus. Message and help are not compared, because
// wording differs between implementations.
func TestErrorCasesMatchCorpus(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "errors", "*", "input.chik"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no error cases: %v", err)
	}
	for _, path := range cases {
		dir := filepath.Dir(path)
		t.Run(filepath.Base(dir), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expectedData, err := os.ReadFile(filepath.Join(dir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			var expected struct {
				Diagnostics []struct {
					Code     string `json:"code"`
					Severity string `json:"severity"`
					Path     string `json:"path"`
					Range    struct {
						Start Pos `json:"start"`
						End   Pos `json:"end"`
					} `json:"range"`
				} `json:"diagnostics"`
			}
			if err := json.Unmarshal(expectedData, &expected); err != nil {
				t.Fatal(err)
			}
			file := Parse(path, string(source))
			// The corpus pins the first diagnostic only: a lexical error makes the
			// rest of the file unreliable, and recovery strategy is not specified.
			if len(file.Diagnostics) < len(expected.Diagnostics) {
				t.Fatalf("got %d diagnostics, want at least %d: %+v", len(file.Diagnostics), len(expected.Diagnostics), file.Diagnostics)
			}
			for index, want := range expected.Diagnostics {
				got := file.Diagnostics[index]
				if got.Severity != want.Severity {
					t.Errorf("diagnostic %d severity = %q, want %q", index, got.Severity, want.Severity)
				}
				if got.Code != want.Code || got.Path != want.Path || got.Range != (Range{Start: want.Range.Start, End: want.Range.End}) {
					t.Errorf("diagnostic %d = %s %q %+v, want %s %q %+v", index, got.Code, got.Path, got.Range, want.Code, want.Path, want.Range)
				}
			}
		})
	}
}
