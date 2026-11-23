package chiklang

import (
	"strings"
	"testing"
)

// TestSchemaTableKeysSurviveRendering proves a table's list and block keys keep
// their shape. Writing them as nested maps lost the key name and produced YAML
// the compiler could not parse.
func TestSchemaTableKeysSurviveRendering(t *testing.T) {
	source := `database {
  profile: "postgresql"
  tables {
    posts {
      columns {
        id: { type: id }
        title: { type: string }
      }
      primary_key: ["id"]
      unique {
        by_title { columns: ["title"] }
      }
      indexes {
        recent { columns: ["title"], unique: false }
      }
    }
  }
}
`
	file := Parse("chik/schema.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	rendered, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	text := string(rendered)
	// Each key must keep its members, not only its name. A name-only check let a
	// rendering that wrote `by_title: columns` pass.
	for _, want := range []string{
		"primary_key: [\"id\"]",
		"by_title:\n          columns: [\"title\"]",
		"recent:\n          columns: [\"title\"]",
		"unique: false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered YAML lacks %q:\n%s", want, text)
		}
	}
	// The broken form wrote a key with no name before the list.
	if strings.Contains(text, "\n        : ") {
		t.Fatalf("table key lost its name:\n%s", text)
	}
}
