package chiklang

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// Change replaces one declaration unit with new source text. An empty Text
// deletes the unit. Path uses the declaration spelling the sync layer already
// uses, such as services.posts.operations.get or auth.methods.
type Change struct {
	Path string
	Text string
}

// EditResult is a complete candidate or the diagnostics that prevented one.
type EditResult struct {
	Source      string
	Diagnostics []Diagnostic
}

// Edit rewrites the named units of a parsed file and returns the formatted
// result. Comments outside the edited units stay attached to their nodes, so
// they survive the rewrite. A file that already has a syntax error is refused
// rather than edited, because its tree does not describe the source.
func Edit(file *File, changes []Change) EditResult {
	if len(file.Diagnostics) != 0 {
		return EditResult{Source: file.Source, Diagnostics: file.Diagnostics}
	}
	if len(changes) == 0 {
		return EditResult{Source: Format(file)}
	}
	spans, diagnostics := editSpans(file, changes)
	if len(diagnostics) != 0 {
		return EditResult{Source: file.Source, Diagnostics: diagnostics}
	}
	edited := applySpans(file.Source, spans)
	parsed := Parse(file.Path, edited)
	if len(parsed.Diagnostics) != 0 {
		return EditResult{Source: file.Source, Diagnostics: parsed.Diagnostics}
	}
	return EditResult{Source: Format(parsed)}
}

// insertion is one new unit appended inside an existing block.
type insertion struct {
	container unit
}

// insertionPoint finds the block that would own a path that does not exist yet.
// The last dotted segment is the new unit's name and the rest is the parent.
func insertionPoint(file *File, path string) (unit, bool) {
	parent := path
	if index := strings.LastIndex(path, "."); index >= 0 {
		parent = path[:index]
	}
	// An operation path names operations explicitly, but the parser attaches
	// operations to their service. The service is the container either way.
	underOperations := strings.HasSuffix(parent, ".operations")
	if underOperations {
		parent = strings.TrimSuffix(parent, ".operations")
	}
	ref, ok := declarationPaths(file)[parent]
	if !ok {
		return unit{}, false
	}
	// The parser flattens an operations block into its service, so there is no
	// node for it. A new operation still belongs inside operations { }, and the
	// block's own brace is found from the service's text.
	if underOperations && ref.node.Kind == NodeService {
		if block, ok := operationsBlock(file, ref); ok {
			return block, true
		}
	}
	if ref.node.Kind != NodeService && ref.node.Kind != NodeServicesBlock {
		return unit{}, false
	}
	return ref, true
}

// operationsBlock returns a synthetic unit spanning the operations { } block of
// a service, so a new operation can be inserted before its closing brace.
func operationsBlock(file *File, service unit) (unit, bool) {
	start, end, ok := unitBytes(file, service)
	if !ok {
		return unit{}, false
	}
	marker := strings.Index(file.Source[start:end], "operations")
	if marker < 0 {
		return unit{}, false
	}
	at := start + marker + len("operations")
	for at < end && (file.Source[at] == ' ' || file.Source[at] == '\t') {
		at++
	}
	if at >= end || file.Source[at] != '{' {
		return unit{}, false
	}
	return unit{node: &Node{Kind: NodeOperation}, start: at}, true
}

// insertionSpan appends a new unit before its container's closing brace, on its
// own line at the container's child indentation.
func insertionSpan(file *File, container unit, text string) span {
	start, end, ok := unitBytes(file, container)
	if !ok {
		return span{start: len(file.Source), end: len(file.Source), text: text}
	}
	// Insert on a fresh line before the container's closing brace. Reusing the
	// indentation of the block's existing entries keeps the result readable even
	// before the formatter runs.
	closing := end - 1
	indent := containerIndent(file.Source, start) + "  "
	if last := previousLineIndent(file.Source, closing); last != "" {
		indent = last
	}
	// The formatter keeps a closing brace that shares a line with the last entry,
	// so the insertion also breaks that line. Without this the new unit lands
	// beside the brace and the result is not canonical.
	// Insert at the end of the last non-blank line before the brace, so the new
	// unit follows the existing entries directly. Inserting at the brace itself
	// leaves the brace's own indentation behind as a blank line.
	end = closing
	for end > start {
		previousIndex := end - 1
		previous := file.Source[previousIndex]
		if previous != '\n' && previous != ' ' && previous != '\t' {
			break
		}
		end--
	}
	return span{start: end, end: end, text: "\n" + indent + text}
}

// previousLineIndent returns the indentation of the last non-empty line before
// a position, which is the indentation the block's entries already use.
func previousLineIndent(source string, at int) string {
	for at > 0 {
		previous := at - 1
		start := lineStart(source, previous)
		line := source[start:at]
		if strings.TrimSpace(line) != "" {
			width := 0
			for width < len(line) && (line[width] == ' ' || line[width] == '\t') {
				width++
			}
			return line[:width]
		}
		at = start
	}
	return ""
}

// containerIndent returns the indentation of the line a block starts on.
func containerIndent(source string, at int) string {
	line := lineStart(source, at)
	width := 0
	for line+width < len(source) && (source[line+width] == ' ' || source[line+width] == '\t') {
		width++
	}
	return source[line : line+width]
}

// span is one byte replacement in the original source.
type span struct {
	start, end int
	text       string
}

// editSpans resolves every change to a byte range, rejecting overlaps so two
// changes cannot claim the same bytes.
func editSpans(file *File, changes []Change) ([]span, []Diagnostic) {
	index := declarationPaths(file)
	spans := make([]span, 0, len(changes))
	var diagnostics []Diagnostic
	for _, change := range changes {
		ref, ok := index[change.Path]
		if !ok {
			// A path that does not exist yet is an addition. It is placed in the
			// container that would own it, so the console can create a new
			// operation instead of only replacing an existing one.
			if parent, ok := insertionPoint(file, change.Path); ok && change.Text != "" {
				spans = append(spans, insertionSpan(file, parent, change.Text))
				continue
			}
			diagnostics = append(diagnostics, Diagnostic{
				Code: CodeInvalidPath, Severity: "error", Path: change.Path,
				Message: "no unit at this path",
			})
			continue
		}
		start, end, ok := unitBytes(file, ref)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Code: CodeInvalidPath, Severity: "error", Path: change.Path,
				Message: "the unit has no source span",
			})
			continue
		}
		text := change.Text
		if text == "" {
			spans = append(spans, deletionSpan(file, start, end))
			continue
		}
		spans = append(spans, span{start: start, end: end, text: text})
	}
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	for i := 1; i < len(spans); i++ {
		if spans[i].start < spans[i-1].end {
			return nil, []Diagnostic{{
				Code: CodeInvalidPath, Severity: "error", Path: "",
				Message: "the requested units overlap",
			}}
		}
	}
	return spans, nil
}

func deletionSpan(file *File, start, end int) span {
	text := ""
	next := sort.Search(len(file.Tokens), func(i int) bool { return file.Tokens[i].Start >= end })
	if next < len(file.Tokens) && file.Tokens[next].Kind == Punct &&
		(file.Tokens[next].Text == ";" || file.Tokens[next].Text == ",") {
		separator := file.Tokens[next]
		text = file.Source[end:separator.Start]
		end = separator.End
	}

	first, last := lineStart(file.Source, start), lineEnd(file.Source, end)
	// Remove an empty line without taking neighboring syntax or comments.
	if strings.TrimSpace(file.Source[first:start]) == "" && strings.TrimSpace(text) == "" &&
		strings.TrimSpace(file.Source[end:last]) == "" {
		return span{start: first, end: last}
	}
	return span{start: start, end: end, text: text}
}

func applySpans(source string, spans []span) string {
	var out strings.Builder
	last := 0
	for _, s := range spans {
		out.WriteString(source[last:s.start])
		out.WriteString(s.text)
		last = s.end
	}
	out.WriteString(source[last:])
	return out.String()
}

// unit is a node together with the byte offset where it starts. A node's Range
// starts at its name token, but a block's closing brace is consumed by the
// parser without being stored in any node's tokens, so the end of the unit has
// to be found in the token stream.
type unit struct {
	node  *Node
	start int
}

// unitBytes returns the byte span a unit occupies, from its first token through
// its closing brace. Leading and trailing trivia stay outside the span, so a
// comment attached to the unit is not rewritten.
func unitBytes(file *File, ref unit) (int, int, bool) {
	if ref.start < 0 || ref.start >= len(file.Source) {
		return 0, 0, false
	}
	start := ref.start
	last := start
	depth := 0
	opened := false
	for _, token := range file.Tokens {
		if token.Start < start {
			continue
		}
		if token.Kind == EOF {
			break
		}
		switch token.Text {
		case "{":
			depth++
			opened = true
		case "}":
			if opened {
				depth--
			}
		}
		if token.End > last {
			last = token.End
		}
		if opened && depth == 0 {
			return start, last, true
		}
	}
	if last <= start {
		return 0, 0, false
	}
	return start, last, true
}

func lineStart(source string, at int) int {
	for at > 0 {
		previous := at - 1
		if source[previous] == '\n' {
			break
		}
		at = previous
	}
	return at
}

func lineEnd(source string, at int) int {
	for at < len(source) && source[at] != '\n' {
		at++
	}
	if at < len(source) {
		at++
	}
	return at
}

// declarationPaths indexes the units a caller may address, using the same
// spelling the settings sync uses so one unit name means one thing in both
// paths. A unit is addressed whole; the sync layer already treats an operation,
// a named type, and a named client as atomic.
func declarationPaths(file *File) map[string]unit {
	index := map[string]unit{}
	root := file.Root
	add := func(path string, node *Node) {
		if node == nil || path == "" {
			return
		}
		if at, ok := offsetOf(file.Source, node.Range.Start); ok {
			index[path] = unit{node: node, start: at}
		}
	}
	for _, block := range root.Children {
		switch block.Kind {
		case NodeProjectBlock:
			projectPaths(file, add, block)
		case NodeServicesBlock:
			for _, service := range block.Children {
				if service.Kind != NodeService {
					continue
				}
				add("services."+service.Name, service)
				for _, operation := range service.Children {
					if operation.Kind == NodeOperation {
						add("services."+service.Name+".operations."+operation.Name, operation)
					}
				}
			}
		case NodeClientsBlock:
			for _, client := range block.Children {
				if client.Name != "" {
					add("clients."+client.Name, client)
				}
			}
		case NodeAuthBlock:
			for _, entry := range block.Children {
				if entry.Name != "" {
					add("auth."+entry.Name, entry)
				}
			}
		case NodeTypesBlock:
			for _, declaration := range block.Children {
				if declaration.Name != "" {
					add("types."+declaration.Name, declaration)
				}
			}
		case NodeSchemaBlock:
			nested(add, "database", block.Children)
		}
	}
	return index
}

// projectPaths maps the project block onto the settings paths YAML uses, so a
// caller does not have to know which file a key lives in.
func projectPaths(file *File, add func(string, *Node), block *Node) {
	for _, entry := range block.Children {
		switch entry.Name {
		case "name":
			add("project.name", entry)
		case "server":
			add("generation.server.language", entry)
		case "clients":
			add("generation.clients", entry)
		case "secrets":
			add("generation.server.secrets", entry)
		case "vault", "realtime":
			add(entry.Name, entry)
		case "storage":
			add("storage", entry)
			if value := child(entry, 0); value != nil {
				for _, pair := range value.Children {
					if pair.Name != "" {
						add("storage."+pair.Name, pair)
					}
				}
			}
		}
	}
}

// nested indexes a block's entries by their dotted path, which is how the
// schema layer names a table, a column, or a key.
func nested(add func(string, *Node), path string, nodes []*Node) {
	for _, node := range nodes {
		if node.Name == "" {
			continue
		}
		add(path+"."+node.Name, node)
		nested(add, path+"."+node.Name, node.Children)
	}
}

// offsetOf converts a one-based line and code-point column into a byte offset.
func offsetOf(source string, pos Pos) (int, bool) {
	if pos.Line < 1 || pos.Column < 1 {
		return 0, false
	}
	line, at := 1, 0
	for line < pos.Line {
		next := strings.IndexByte(source[at:], '\n')
		if next < 0 {
			return 0, false
		}
		at += next + 1
		line++
	}
	column := 1
	for at < len(source) && column < pos.Column {
		_, width := utf8.DecodeRuneInString(source[at:])
		at += width
		column++
	}
	if at > len(source) {
		return 0, false
	}
	return at, true
}
