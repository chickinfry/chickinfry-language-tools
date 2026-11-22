package chiklang

import (
	_ "embed"
	"fmt"
	"sort"
)

// The examples are copies of the conformance corpus cases in
// this repository, which is the specification. They are
// copied rather than written here so a skill or a document cannot print a
// declaration the parser rejects: the corpus cases are the ones every
// implementation must pass.
//
//go:embed examples/project.chik
var projectExample string

//go:embed examples/schema.chik
var schemaExample string

//go:embed examples/api.chik
var apiExample string

//go:embed examples/clients.chik
var clientsExample string

//go:embed examples/auth.chik
var authExample string

// Example is one role file's declaration example.
type Example struct {
	// Role is the declaration role: project, schema, api, clients, or auth.
	Role string
	// Path is where the file lives in a project.
	Path string
	// Source is the example, ending with one newline.
	Source string
}

var examples = []Example{
	{Role: "project", Path: "chik/project.chik", Source: projectExample},
	{Role: "schema", Path: "chik/schema.chik", Source: schemaExample},
	{Role: "api", Path: "chik/api.chik", Source: apiExample},
	{Role: "clients", Path: "chik/clients.chik", Source: clientsExample},
	{Role: "auth", Path: "chik/auth.chik", Source: authExample},
}

// Examples returns the declaration examples in role order, each copied from a
// conformance case. A caller that prints them is printing the specification.
func Examples() []Example {
	ordered := append([]Example(nil), examples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Role < ordered[j].Role })
	return ordered
}

// ExampleFor returns one role's example, or an error naming the roles that
// exist. A role with no example is a programming error, not a user error.
func ExampleFor(role string) (Example, error) {
	for _, example := range examples {
		if example.Role == role {
			return example, nil
		}
	}
	return Example{}, fmt.Errorf("no declaration example for role %q", role)
}
