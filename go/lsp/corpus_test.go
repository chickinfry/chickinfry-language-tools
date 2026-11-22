package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func corpusRoot(t *testing.T) string {
	t.Helper()
	root := "../.."
	if _, err := os.Stat(filepath.Join(root, "conformance")); err != nil {
		t.Fatalf("read the conformance corpus: %v", err)
	}
	return root
}

type definitionCase struct {
	Position struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"position"`
	Targets []struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"targets"`
}

type completionCase struct {
	Position struct {
		Line   int `json:"line"`
		Column int `json:"column"`
	} `json:"position"`
	Includes []string `json:"includes"`
}

// TestDefinitionCases proves the public gate: a name at a position resolves to
// the declaration the corpus names. The corpus is the specification, so this
// is the same case an independent implementation must pass.
func TestDefinitionCases(t *testing.T) {
	root := corpusRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "conformance", "definition", "*", "input.chik"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no definition cases: %v", err)
	}
	for _, path := range paths {
		name := filepath.Base(filepath.Dir(path))
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expectedPath := filepath.Join(filepath.Dir(path), "expected.json")
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}
			var expected definitionCase
			if err := json.Unmarshal(expectedBytes, &expected); err != nil {
				t.Fatal(err)
			}

			s := &server{documents: map[string]*document{}}
			uri := "file:///project/chik/api.chik"
			s.documents[uri] = newDocument(uri, 1, string(source))
			doc := s.documents[uri]
			// The corpus is 1-based code points; the protocol is 0-based UTF-16.
			position := doc.position(expected.Position.Line, expected.Position.Column)
			word := doc.wordAt(position)
			if word == "" {
				t.Fatalf("no word at line %d column %d", expected.Position.Line, expected.Position.Column)
			}
			locations := s.declarations(word, uri)

			var got [][2]int
			for _, location := range locations {
				got = append(got, [2]int{location.Range.Start.Line + 1, location.Range.Start.Character + 1})
			}
			want := make([][2]int, 0, len(expected.Targets))
			for _, target := range expected.Targets {
				want = append(want, [2]int{target.Line, target.Column})
			}
			sortPairs(got)
			sortPairs(want)
			if len(got) != len(want) {
				t.Fatalf("definition of %q = %v, want %v", word, got, want)
			}
			for index := range want {
				if got[index] != want[index] {
					t.Fatalf("definition of %q = %v, want %v", word, got, want)
				}
			}
		})
	}
}

// TestCompletionCases proves the other public gate. Only the listed labels are
// required; a server may offer more.
func TestCompletionCases(t *testing.T) {
	root := corpusRoot(t)
	paths, err := filepath.Glob(filepath.Join(root, "conformance", "completion", "*", "input.chik"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no completion cases: %v", err)
	}
	for _, path := range paths {
		name := filepath.Base(filepath.Dir(path))
		t.Run(name, func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expectedBytes, err := os.ReadFile(filepath.Join(filepath.Dir(path), "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			var expected completionCase
			if err := json.Unmarshal(expectedBytes, &expected); err != nil {
				t.Fatal(err)
			}

			s := &server{documents: map[string]*document{}}
			uri := "file:///project/chik/api.chik"
			s.documents[uri] = newDocument(uri, 1, string(source))

			labels := map[string]bool{}
			for _, item := range s.completionItems() {
				labels[item.Label] = true
			}
			for _, want := range expected.Includes {
				if !labels[want] {
					t.Errorf("completion is missing %q; offered %v", want, sortedLabels(labels))
				}
			}
		})
	}
}

func sortPairs(pairs [][2]int) {
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i][0] != pairs[j][0] {
			return pairs[i][0] < pairs[j][0]
		}
		return pairs[i][1] < pairs[j][1]
	})
}

func sortedLabels(labels map[string]bool) []string {
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
