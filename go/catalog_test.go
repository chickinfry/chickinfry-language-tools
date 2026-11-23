package chiklang

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedTableMatchesCorpus keeps the embedded copy from drifting away
// from the published specification. The copy exists so chik explain works without a
// network fetch, and a silent difference would describe a different problem.
func TestEmbeddedTableMatchesCorpus(t *testing.T) {
	root := corpusRoot(t)
	want, err := os.ReadFile(filepath.Join(root, "spec", "error-codes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != publishedCodes {
		t.Fatal("go/spec/error-codes.md differs from spec/error-codes.md")
	}
}
