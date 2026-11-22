package chiklang

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRendererCoversEveryErrorCase runs the human and JSON forms over each
// corpus error input. The corpus pins which diagnostics exist; this checks that
// the renderer produces a complete, decodable form for all of them, including
// a file whose only error is an unterminated string or comment.
func TestRendererCoversEveryErrorCase(t *testing.T) {
	root := corpusRoot(t)
	cases, err := filepath.Glob(filepath.Join(root, "conformance", "errors", "*", "input.chik"))
	if err != nil || len(cases) == 0 {
		t.Fatalf("no error cases: %v", err)
	}
	for _, path := range cases {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			file := Parse(filepath.ToSlash(filepath.Join("chik", filepath.Base(path))), string(source))
			if len(file.Diagnostics) == 0 {
				t.Fatal("the case produced no diagnostic")
			}

			rendered := Render(file)
			if strings.Count(rendered, "= help: ") != len(file.Diagnostics) {
				t.Fatalf("every diagnostic needs a help line:\n%s", rendered)
			}
			if strings.Contains(rendered, "\r") {
				t.Fatalf("rendered form keeps a carriage return:\n%q", rendered)
			}

			data, err := JSON(file)
			if err != nil {
				t.Fatal(err)
			}
			var report struct {
				Version     int    `json:"version"`
				File        string `json:"file"`
				Diagnostics []struct {
					Code    string   `json:"code"`
					Help    *string  `json:"help"`
					Notes   []string `json:"notes"`
					Range   Range    `json:"range"`
					Message string   `json:"message"`
				} `json:"diagnostics"`
			}
			if err := json.Unmarshal(data, &report); err != nil {
				t.Fatalf("JSON form does not decode: %v\n%s", err, data)
			}
			if report.Version != JSONVersion || len(report.Diagnostics) != len(file.Diagnostics) {
				t.Fatalf("report = %s", data)
			}
			for _, diagnostic := range report.Diagnostics {
				if diagnostic.Help == nil || diagnostic.Message == "" {
					t.Fatalf("diagnostic is missing help or message: %s", data)
				}
				if diagnostic.Notes == nil {
					t.Fatalf("notes must be an array, not null: %s", data)
				}
				if diagnostic.Range.Start.Line < 1 || diagnostic.Range.Start.Column < 1 {
					t.Fatalf("range is not one-based: %s", data)
				}
			}
		})
	}
}
