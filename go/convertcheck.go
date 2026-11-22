package chiklang

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// refusal reports a declaration the converter cannot express without loss.
// Conversion is not allowed to drop a key: a lost default, a lost one_of
// group, or an unknown field would change the declaration silently, and the
// converted file would look complete.
type refusal struct {
	path    string
	message string
}

// checkConvertible walks the document and collects every construct with no
// .chik form. It runs before rendering so a refusal never leaves a partial
// result.
func checkConvertible(role string, node *yaml.Node) []refusal {
	c := &convertCheck{role: role}
	c.walk(node, "")
	return c.refusals
}

type convertCheck struct {
	role     string
	refusals []refusal
}

func (c *convertCheck) refuse(path, message string) {
	c.refusals = append(c.refusals, refusal{path: path, message: message})
}

// walk visits every mapping and sequence, refusing a construct the grammar has
// no form for. It deliberately checks named constructs rather than every key:
// a key may be a declaration name, and a name is not a field.
func (c *convertCheck) walk(node *yaml.Node, path string) {
	if node == nil {
		return
	}
	switch node.Kind {
	case yaml.MappingNode:
		for _, entry := range pairs(node) {
			child := path + "." + entry.name
			if path == "" {
				child = entry.name
			}
			c.checkKey(entry.name, child, entry.node)
			c.walk(entry.node, child)
		}
	case yaml.SequenceNode:
		for index, item := range node.Content {
			c.walk(item, fmt.Sprintf("%s[%d]", path, index))
		}
	case yaml.AliasNode:
		c.refuse(path, "an alias has no .chik form")
	}
}

// checkKey refuses the keys the beta grammar cannot express.
func (c *convertCheck) checkKey(name, path string, value *yaml.Node) {
	switch name {
	case "one_of":
		c.refuse(path, "one_of groups have no .chik form yet; keep this declaration in YAML")
	case "default":
		// A database column default has a form; an API input default does not.
		if c.role != "schema" {
			c.refuse(path, "an input default has no .chik form yet; keep this declaration in YAML")
		}
	case "offline", "method", "public":
		if value.Kind != yaml.ScalarNode || value.Value != "false" || name == "public" {
			c.refuse(path, name+" has no .chik form yet; keep this declaration in YAML")
		}
	}
}
