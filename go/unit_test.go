package chiklang

import (
	"strings"
	"testing"
)

// TestRoleForPathAgreesWithTheWriter keeps one path from naming two files.
func TestRoleForPathAgreesWithTheWriter(t *testing.T) {
	for path, want := range map[string]string{
		"services.posts.operations.get": "api",
		"types.Post":                    "api",
		"auth.methods":                  "auth",
		"clients.web":                   "clients",
		"storage.assets":                "project",
		"generation.clients":            "project",
		"vault":                         "project",
		"database.tables.posts":         "schema",
	} {
		if got := RoleForPath(path); got != want {
			t.Fatalf("RoleForPath(%q) = %q, want %q", path, got, want)
		}
	}
	if got := RoleForPath("unrelated"); got != "" {
		t.Fatalf("unknown path mapped to %q", got)
	}
}

// TestUnitTextRendersOnlyTheRequestedUnits proves a pull writes the changed
// units from the merged declaration instead of re-serializing by hand.
func TestUnitTextRendersOnlyTheRequestedUnits(t *testing.T) {
	doc := []byte("services:\n  posts:\n    operations:\n      get:\n        input:\n          id: {type: id}\n        output:\n          title: {type: string}\n      list:\n        output:\n          title: {type: string}\n")
	units, diagnostics := UnitText("api", doc, []string{"services.posts.operations.list"})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	got, ok := units["services.posts.operations.list"]
	if !ok {
		t.Fatalf("unit missing: %v", units)
	}
	if !strings.HasPrefix(got, "list()") {
		t.Fatalf("unit text = %q", got)
	}
	if _, extra := units["services.posts.operations.get"]; extra {
		t.Fatal("an unrequested unit was rendered")
	}
}

// TestUnitTextAppliesToARealFile proves the rendered unit replaces the old one
// and leaves the rest of the file, including comments, in place.
func TestUnitTextAppliesToARealFile(t *testing.T) {
	source := "// kept\nservices {\n  posts {\n    operations {\n      get(id: id) -> { title: string }\n    }\n  }\n}\n"
	doc := []byte("services:\n  posts:\n    operations:\n      get:\n        input:\n          id: {type: id}\n        output:\n          title: {type: string}\n          body: {type: string}\n")
	units, diagnostics := UnitText("api", doc, []string{"services.posts.operations.get"})
	if len(diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", diagnostics)
	}
	result := Edit(Parse("chik/api.chik", source), []Change{{Path: "services.posts.operations.get", Text: units["services.posts.operations.get"]}})
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", result.Diagnostics)
	}
	if !strings.Contains(result.Source, "// kept") {
		t.Fatalf("comment lost:\n%s", result.Source)
	}
	if !strings.Contains(result.Source, "body: string") {
		t.Fatalf("edit did not apply:\n%s", result.Source)
	}
}
