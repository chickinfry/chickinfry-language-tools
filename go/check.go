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

// Check validates names, references, cycles, and modifiers that depend on
// where a named type is used.
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

// types resolves named references, checks for cycles, and validates output uses.
func (c *checker) types() {
	graph := map[string][]string{}
	optionalTypes := map[string]bool{}
	var outputRefs []*Node
	var walk func(node *Node, output bool)
	walk = func(node *Node, output bool) {
		output = output || node.Kind == NodeResult
		if node.Optional && c.currentType != "" {
			optionalTypes[c.currentType] = true
		}
		if node.Kind == NodeTypeRef && !builtinTypes[node.Name] && !isKeywordType(node.Name) {
			if !c.declared[node.Name] {
				c.report(CodeUnknownType, node.Path, node, "unknown type")
			} else if current := c.currentType; current != "" {
				graph[current] = append(graph[current], node.Name)
			} else if output {
				outputRefs = append(outputRefs, node)
			}
		}
		for _, child := range node.Children {
			walk(child, output)
		}
	}
	for _, block := range c.file.Root.Children {
		switch block.Kind {
		case NodeTypesBlock:
			for _, decl := range block.Children {
				c.currentType = decl.Name
				for _, child := range decl.Children {
					walk(child, false)
				}
			}
		case NodeServicesBlock:
			// An operation names types too, and an unknown one must not wait for
			// the generator to fail.
			c.currentType = ""
			walk(block, false)
		}
	}
	c.currentType = ""
	for name := range graph {
		if reachable(graph, name, func(next string) bool { return next == name }, map[string]bool{}) {
			c.report(CodeCyclicType, "types."+name, nil, "a named type cannot refer to itself")
		}
	}
	if len(optionalTypes) == 0 {
		return
	}
	for _, ref := range outputRefs {
		hasOptional, checked := optionalTypes[ref.Name]
		if !checked {
			hasOptional = reachable(graph, ref.Name, func(next string) bool { return optionalTypes[next] }, map[string]bool{})
			optionalTypes[ref.Name] = hasOptional
		}
		if hasOptional {
			c.reportAt(CodeInvalidModifier, ref.Path, ref.Range, "this type has optional fields and can only be used as input")
		}
	}
}

func reachable(graph map[string][]string, from string, matches func(string) bool, seen map[string]bool) bool {
	for _, next := range graph[from] {
		if matches(next) {
			return true
		}
		if seen[next] {
			continue
		}
		seen[next] = true
		if reachable(graph, next, matches, seen) {
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

// reportAt reports a diagnostic at an explicit source range.
func (c *checker) reportAt(code, path string, at Range, message string) {
	c.diags = append(c.diags, Diagnostic{Code: code, Severity: "error", Path: path, Message: message, Range: at})
}
