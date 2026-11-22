package chiklang

import (
	"strings"
	"testing"
)

// TestConvertEveryRoleParses converts a realistic document per role and checks
// the result parses. A converter that emits something the language rejects is
// worse than one that refuses, because the failure surfaces later.
func TestConvertEveryRoleParses(t *testing.T) {
	cases := map[string]string{
		"schema": `database:
  profile: postgresql
  tables:
    posts:
      columns:
        id: {type: id, default: {kind: uuid}}
        title: {type: string}
      primary_key: [id]
      indexes:
        by_title: {columns: [title]}
`,
		"auth": `auth:
  methods: [email-password]
  origins: ["https://app.example.test"]
  signup: open
  browser_session: false
  session:
    idle_timeout_seconds: 2592000
`,
		"clients": `clients:
  web:
    template: typescript
    api: full
`,
	}
	// Legacy extensions are migrated into project.chik as capabilities.
	if _, diagnostics := Convert("extensions", []byte("extensions:\n  - id: platform/control-plane\n")); len(diagnostics) == 0 {
		t.Fatal("Convert accepted the retired extensions role")
	}
	fragment, diagnostics := ConvertExtensions([]byte("extensions:\n  - id: platform/control-plane\n"))
	if len(diagnostics) != 0 {
		t.Fatalf("extensions refused: %+v", diagnostics)
	}
	if !strings.Contains(fragment, "capabilities:") || !strings.Contains(fragment, "control_plane") {
		t.Fatalf("extensions fragment = %q", fragment)
	}
	if strings.Contains(fragment, "extension") {
		t.Fatalf("the key must not use the retired word: %q", fragment)
	}

	for role, source := range cases {
		t.Run(role, func(t *testing.T) {
			converted, diagnostics := Convert(role, []byte(source))
			if len(diagnostics) != 0 {
				t.Fatalf("refused: %+v", diagnostics)
			}
			file := Parse("chik/"+role+".chik", converted)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("converted output does not parse: %+v\n%s", file.Diagnostics, converted)
			}
			if !strings.HasPrefix(converted, blockName(role)+" {") {
				t.Fatalf("unexpected opening: %s", converted)
			}
		})
	}
}

func blockName(role string) string {
	if role == "schema" {
		return "database"
	}
	return role
}
