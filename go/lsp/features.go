package lsp

import (
	"io"
	"sort"

	"github.com/chickinfry/chik-lang/go"
)

// documentPosition is the shared params shape for definition and completion.
type documentPosition struct {
	TextDocument struct {
		URI string `json:"uri"`
	} `json:"textDocument"`
	Position Position `json:"position"`
}

// definition returns the declaration a name at the cursor points at. The CST
// already carries every declaration name and its range, so this is a lookup,
// not a second parse.
func (s *server) definition(out io.Writer, message *Message) error {
	var params documentPosition
	if err := mustMessage(message, &params); err != nil {
		return s.reply(out, message.ID, nil)
	}
	doc := s.documents[params.TextDocument.URI]
	if doc == nil {
		return s.reply(out, message.ID, nil)
	}
	word := doc.wordAt(params.Position)
	if word == "" {
		return s.reply(out, message.ID, nil)
	}
	found := s.declarations(word, params.TextDocument.URI)
	if len(found) == 0 {
		return s.reply(out, message.ID, nil)
	}
	// Every target is returned, with the current document first: a name can be
	// declared in more than one open file, and the local one is what the user
	// meant. LSP accepts a Location or an array; an array keeps all of them.
	return s.reply(out, message.ID, found)
}

// declarations finds the files that declare name. The current document comes
// first so a local declaration wins over one in another file.
func (s *server) declarations(name, current string) []Location {
	var found []Location
	for _, uri := range s.documentURIs(current) {
		doc := s.documents[uri]
		file := chiklang.Parse(pathOf(uri), doc.text)
		for _, declaration := range declaredNames(file.Root) {
			if declaration.name != name {
				continue
			}
			found = append(found, Location{
				URI: uri,
				Range: Range{
					Start: doc.position(declaration.at.Start.Line, declaration.at.Start.Column),
					End:   doc.position(declaration.at.End.Line, declaration.at.End.Column),
				},
			})
		}
	}
	return found
}

// documentURIs lists the open documents with current first, then the rest in a
// stable order. A caller that passes an empty current gets the plain order.
func (s *server) documentURIs(current string) []string {
	uris := make([]string, 0, len(s.documents))
	for uri := range s.documents {
		if uri == current {
			continue
		}
		uris = append(uris, uri)
	}
	sort.Strings(uris)
	if _, ok := s.documents[current]; ok {
		return append([]string{current}, uris...)
	}
	return uris
}

type declaration struct {
	name string
	// at is the declaration name, not the whole node, so go-to-definition
	// lands on the name a reader is looking for.
	at chiklang.Range
}

// declaredNames walks the tree for the names a user can navigate to: a type, a
// service, an operation, a parameter, a field, and a config entry.
func declaredNames(node *chiklang.Node) []declaration {
	if node == nil {
		return nil
	}
	var found []declaration
	if navigable(node.Kind) && node.Name != "" && isDeclaration(node) {
		at := node.NameRange
		if at.End.Line == 0 {
			at = node.Range
		}
		found = append(found, declaration{name: node.Name, at: at})
	}
	for _, child := range node.Children {
		found = append(found, declaredNames(child)...)
	}
	return found
}

// isDeclaration separates a declaration from a use. The parser stores both as
// a NodeTypeRef: a declaration carries the name range it was declared at, a
// reference carries only the token it was read from. Without this check a
// definition would resolve a name to itself.
func isDeclaration(node *chiklang.Node) bool {
	if node.Kind != chiklang.NodeTypeRef {
		return true
	}
	return node.NameRange.End.Line != 0
}

func navigable(kind chiklang.NodeKind) bool {
	switch kind {
	case chiklang.NodeTypeRef, chiklang.NodeService, chiklang.NodeOperation,
		chiklang.NodeParam, chiklang.NodeField, chiklang.NodeEntry, chiklang.NodeTable,
		chiklang.NodeColumn:
		return true
	}
	return false
}

// completion offers the names the language defines plus the ones the open
// files declare. A client filters as the user types, so the list stays small.
func (s *server) completion(out io.Writer, message *Message) error {
	var params documentPosition
	if err := mustMessage(message, &params); err != nil {
		return s.reply(out, message.ID, CompletionList{Items: []CompletionItem{}})
	}
	doc := s.documents[params.TextDocument.URI]
	if doc == nil {
		return s.reply(out, message.ID, CompletionList{Items: []CompletionItem{}})
	}
	return s.reply(out, message.ID, CompletionList{Items: s.completionItems()})
}

// completionItems is the label set at any position: the words the grammar
// fixes plus every name the open files declare. A client filters by prefix, so
// the list does not depend on the cursor beyond which document is open.
func (s *server) completionItems() []CompletionItem {
	items := builtinItems()
	seen := map[string]bool{}
	for _, item := range items {
		seen[item.Label] = true
	}
	for _, uri := range s.documentURIs("") {
		other := s.documents[uri]
		file := chiklang.Parse(pathOf(uri), other.text)
		for _, declaration := range declaredNames(file.Root) {
			if seen[declaration.name] {
				continue
			}
			seen[declaration.name] = true
			items = append(items, CompletionItem{Label: declaration.name, Kind: completionKind(declaration.name), Detail: "declared in " + pathOf(uri)})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// completionKind maps a declaration to an LSP CompletionItemKind. A type is
// UpperCamelCase by rule, which is enough to tell it from a name.
func completionKind(name string) int {
	if name != "" && name[0] >= 'A' && name[0] <= 'Z' {
		return 22 // Struct
	}
	return 3 // Function
}

// builtinItems are the words the grammar fixes, so they complete even in an
// empty project.
func builtinItems() []CompletionItem {
	words := []struct {
		label  string
		detail string
	}{
		{"id", "built-in type"},
		{"string", "built-in type"},
		{"timestamp", "built-in type"},
		{"json", "built-in type"},
		{"types", "block"},
		{"services", "block"},
		{"operations", "section"},
		{"project", "block"},
		{"database", "block"},
		{"auth", "block"},
		{"clients", "block"},
	}
	items := make([]CompletionItem, 0, len(words))
	for _, word := range words {
		items = append(items, CompletionItem{Label: word.label, Kind: 14, Detail: word.detail})
	}
	return items
}

// formatting replaces the whole document with the formatter output. A full
// range is simpler than a diff, and the formatter is idempotent, so the
// editor does not fight the user.
func (s *server) formatting(out io.Writer, message *Message) error {
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
		Options map[string]any `json:"options"`
	}
	if err := mustMessage(message, &params); err != nil {
		return s.reply(out, message.ID, []TextEdit{})
	}
	doc := s.documents[params.TextDocument.URI]
	if doc == nil {
		return s.reply(out, message.ID, []TextEdit{})
	}
	file := chiklang.Parse(pathOf(params.TextDocument.URI), doc.text)
	// A document with diagnostics may have lost bytes in the lexer, so its
	// formatter output would delete what the user typed. Leave it alone.
	formatted, ok := chiklang.FormatChecked(file)
	if !ok || formatted == doc.text {
		return s.reply(out, message.ID, []TextEdit{})
	}
	return s.reply(out, message.ID, []TextEdit{{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   doc.position(len(doc.lines), len([]rune(doc.lines[len(doc.lines)-1]))+1),
		},
		NewText: formatted,
	}})
}
