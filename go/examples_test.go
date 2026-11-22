package chiklang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every embedded example must parse without a diagnostic. A skill that printed
// a declaration the parser rejects would teach a user to write one.
func TestExamplesParseCleanly(t *testing.T) {
	for _, example := range Examples() {
		t.Run(example.Role, func(t *testing.T) {
			file := Parse(example.Path, example.Source)
			for _, diagnostic := range file.Diagnostics {
				t.Errorf("%s %s %+v", diagnostic.Code, diagnostic.Path, diagnostic.Range)
			}
			if got := Serialize(file); got != example.Source {
				t.Errorf("example does not round-trip\nwant:\n%s\ngot:\n%s", example.Source, got)
			}
			if got := Format(file); got != example.Source {
				t.Errorf("example is not canonical\nwant:\n%s\ngot:\n%s", example.Source, got)
			}
		})
	}
}

// The examples are copies of conformance cases. When the corpus is checked
// out, this proves the copy still matches the case it came from, so a corpus
// change cannot leave a stale example in the skill.
func TestExamplesMatchTheCorpus(t *testing.T) {
	root := corpusRoot(t)
	cases := map[string]string{
		"project": "convert/plain_project/expected/project.chik",
		"schema":  "convert/schema_columns/expected/schema.chik",
		"api":     "convert/api_scalar_operation/expected/api.chik",
		"clients": "convert/clients_named_apps/expected/clients.chik",
		"auth":    "convert/auth_policy/expected/auth.chik",
	}
	for role, relative := range cases {
		t.Run(role, func(t *testing.T) {
			example, err := ExampleFor(role)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "conformance", relative)
			corpus, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if strings.TrimRight(string(corpus), "\n") != strings.TrimRight(example.Source, "\n") {
				t.Fatalf("the %s example drifted from %s", role, relative)
			}
		})
	}
}

// Every role file has an example, and no example names a role the project
// does not have. The roles decision fixed five files and retired extensions.
func TestExamplesCoverTheFiveRoleFiles(t *testing.T) {
	want := []string{"api", "auth", "clients", "project", "schema"}
	got := make([]string, 0, len(Examples()))
	for _, example := range Examples() {
		got = append(got, example.Role)
		if !strings.HasSuffix(example.Path, "/"+example.Role+".chik") {
			t.Errorf("path %q does not name role %q", example.Path, example.Role)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("roles = %v, want %v", got, want)
	}
}
