package lsp

import (
	"testing"
)

// A name declared in two open files returns both targets, with the document
// the request came from first. The corpus gate only sees one target, so this
// is the case it cannot catch.
func TestDefinitionReturnsEveryTargetCurrentFirst(t *testing.T) {
	declaration := "types {\n  Post = { id: id }\n}\n"
	use := "services {\n  posts {\n    operations {\n      get() -> { post: Post }\n    }\n  }\n}\n"

	s := &server{documents: map[string]*document{}}
	// The alphabetically later URI is the current one, so a plain sort would
	// return the wrong document first.
	s.documents["file:///p/a-extra.chik"] = newDocument("file:///p/a-extra.chik", 1, declaration)
	s.documents["file:///p/z-main.chik"] = newDocument("file:///p/z-main.chik", 1, declaration+use)

	current := s.documents["file:///p/z-main.chik"]
	position := current.position(7, 24)
	word := current.wordAt(position)
	if word != "Post" {
		t.Fatalf("word at the cursor = %q, want Post", word)
	}

	found := s.declarations(word, "file:///p/z-main.chik")
	if len(found) != 2 {
		t.Fatalf("found %d targets, want 2: %+v", len(found), found)
	}
	if found[0].URI != "file:///p/z-main.chik" {
		t.Errorf("first target is %s, want the current document", found[0].URI)
	}
}

// With one declaration the array has one entry, which is what the corpus
// gate checks.
func TestDefinitionWithOneTarget(t *testing.T) {
	source := "types {\n  Post = { id: id }\n}\nservices {\n  posts {\n    operations {\n      get() -> { post: Post }\n    }\n  }\n}\n"
	uri := "file:///p/chik/api.chik"
	s := &server{documents: map[string]*document{}}
	s.documents[uri] = newDocument(uri, 1, source)
	doc := s.documents[uri]
	found := s.declarations(doc.wordAt(doc.position(7, 24)), uri)
	if len(found) != 1 {
		t.Fatalf("found %d targets, want 1: %+v", len(found), found)
	}
	if found[0].Range.Start.Line != 1 {
		t.Errorf("target line = %d, want 1 (zero-based, the Post declaration)", found[0].Range.Start.Line)
	}
}
