package chiklang

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

// commentLines renders the comments attached to a YAML node as .chik comment
// lines at the given indent. A leading comment becomes // lines above the
// entry, and a same-line comment follows the entry.
//
// Comments are why a conversion is worth doing by hand once: a declaration
// carries the reasoning for a field, and dropping it makes the converted file
// poorer than the one it replaced.
func commentLines(node *yaml.Node, indent string) string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	for _, line := range strings.Split(node.HeadComment, "\n") {
		if text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#")); text != "" {
			out.WriteString(indent + "// " + text + "\n")
		}
	}
	return out.String()
}

// trailingComment renders a same-line comment, or "" when there is none.
func trailingComment(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(node.LineComment), "#"))
	if text == "" {
		return ""
	}
	return " // " + text
}

// footComments renders comments that follow a node on their own lines.
func footComments(node *yaml.Node, indent string) string {
	if node == nil {
		return ""
	}
	var out strings.Builder
	for _, line := range strings.Split(node.FootComment, "\n") {
		if text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#")); text != "" {
			out.WriteString(indent + "// " + text + "\n")
		}
	}
	return out.String()
}
