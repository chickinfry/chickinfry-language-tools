package chiklang

import (
	"regexp"
	"strings"
)

var (
	lowerSnake = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	upperCamel = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
)

// builtinTypes are the types a declaration may name without declaring them.
var builtinTypes = map[string]bool{"id": true, "string": true, "timestamp": true, "json": true}

// Check runs the checks that need the whole file: name case, unknown types,
// and cycles. It returns the diagnostics in source order.
func Check(file *File) []Diagnostic {
	c := &checker{file: file, declared: map[string]bool{}, position: map[string]Token{}}
	c.collect()
	c.names()
	c.types()
	c.capabilities()
	return c.diags
}

// capabilities rejects a repeated key in the capabilities map. The key is the
// capability's name, so a repeat would silently pick one of two identifiers.
func (c *checker) capabilities() {
	for _, block := range c.file.Root.Children {
		if block.Kind != NodeProjectBlock {
			continue
		}
		for _, entry := range block.Children {
			if entry.Name != "capabilities" {
				continue
			}
			seen := map[string]bool{}
			for _, name := range child(entry, 0).Children {
				if seen[name.Name] {
					c.report(CodeDuplicateName, "project.capabilities."+name.Name, name, "this capability name appears more than once")
					continue
				}
				seen[name.Name] = true
			}
		}
	}
}

type checker struct {
	file        *File
	declared    map[string]bool
	position    map[string]Token
	diags       []Diagnostic
	currentType string
}

// collect records every named type so a reference can be resolved later.
func (c *checker) collect() {
	for _, block := range c.file.Root.Children {
		if block.Kind != NodeTypesBlock {
			continue
		}
		for _, decl := range block.Children {
			c.declared[decl.Name] = true
			if token := firstToken(decl); token != nil {
				c.position[decl.Name] = *token
			}
		}
	}
}

// names checks the case rule each position requires.
func (c *checker) names() {
	for _, block := range c.file.Root.Children {
		if block.Kind != NodeTypesBlock {
			continue
		}
		for _, decl := range block.Children {
			if !upperCamel.MatchString(decl.Name) {
				c.reportAt(CodeInvalidName, decl.Path, decl.NameRange, "type names use UpperCamelCase")
			}
		}
	}
}

// types resolves every named reference and rejects a cycle.
func (c *checker) types() {
	graph := map[string][]string{}
	var walk func(node *Node)
	walk = func(node *Node) {
		if node.Kind == NodeTypeRef && !builtinTypes[node.Name] && !isKeywordType(node.Name) {
			if !c.declared[node.Name] {
				c.report(CodeUnknownType, node.Path, node, "unknown type")
			} else if current := c.currentType; current != "" {
				graph[current] = append(graph[current], node.Name)
			}
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	for _, block := range c.file.Root.Children {
		switch block.Kind {
		case NodeTypesBlock:
			for _, decl := range block.Children {
				c.currentType = decl.Name
				for _, child := range decl.Children {
					walk(child)
				}
			}
		case NodeServicesBlock:
			// An operation names types too, and an unknown one must not wait for
			// the generator to fail.
			c.currentType = ""
			walk(block)
		}
	}
	c.currentType = ""
	for name := range graph {
		if reachable(graph, name, name, map[string]bool{}) {
			c.report(CodeCyclicType, "types."+name, nil, "a named type cannot refer to itself")
		}
	}
}

func reachable(graph map[string][]string, from, target string, seen map[string]bool) bool {
	for _, next := range graph[from] {
		if next == target {
			return true
		}
		if seen[next] {
			continue
		}
		seen[next] = true
		if reachable(graph, next, target, seen) {
			return true
		}
	}
	return false
}

func isKeywordType(name string) bool {
	switch name {
	case "Page", "map", "union":
		return true
	}
	return false
}

func firstToken(node *Node) *Token {
	if len(node.Tokens) != 0 {
		return &node.Tokens[0]
	}
	for _, child := range node.Children {
		if token := firstToken(child); token != nil {
			return token
		}
	}
	return nil
}

func (c *checker) report(code, path string, node *Node, message string) {
	var token Token
	if node != nil {
		if found := firstToken(node); found != nil {
			token = *found
		} else {
			token = Token{Line: node.Range.Start.Line, Column: node.Range.Start.Column}
		}
	} else if found, ok := c.position[strings.TrimPrefix(path, "types.")]; ok {
		token = found
	}
	end := Pos{Line: token.Line, Column: token.Column + max(1, token.width())}
	c.diags = append(c.diags, Diagnostic{
		Code: code, Severity: "error", Path: path, Message: message,
		Range: Range{Start: Pos{Line: token.Line, Column: token.Column}, End: end},
	})
}

// reportAt reports at an explicit range, for a diagnostic whose position is the
// declaration name rather than the first token of a value.
func (c *checker) reportAt(code, path string, at Range, message string) {
	c.diags = append(c.diags, Diagnostic{Code: code, Severity: "error", Path: path, Message: message, Range: at})
}
