package chiklang

import (
	"strings"
	"testing"
)

// TestUnknownEntryIsRefusedNotDropped keeps a misspelled or unsupported field
// from compiling into a project that means something else. The YAML form
// rejects an unknown field, so the .chik form must not lose it quietly.
func TestUnknownEntryIsRefusedNotDropped(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"project", "project {\n  name: \"demo\"\n  nonsense: \"x\"\n}\n", "nonsense"},
		{"database", "database {\n  profile: \"postgresql\"\n  nonsense: \"x\"\n}\n", "nonsense"},
		{"auth", "auth {\n  methods: [\"email-password\"]\n  nonsense: \"x\"\n}\n", "nonsense"},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := Parse("chik/"+test.name+".chik", test.source)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("the file should parse: %+v", file.Diagnostics)
			}
			_, err := ToYAML(file)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unknown entry was accepted: %v", err)
			}
			if !strings.Contains(err.Error(), "chik/"+test.name+".chik:") {
				t.Fatalf("error has no file and position:\n%v", err)
			}
		})
	}
}

// TestKnownEntriesStillConvert keeps the refusal from swallowing the entries
// the beta does map.
func TestKnownEntriesStillConvert(t *testing.T) {
	source := "project {\n  name: \"demo\"\n  server: \"typescript\"\n  clients: [\"typescript\"]\n}\n"
	file := Parse("chik/project.chik", source)
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"project:", "name: \"demo\"", "language: \"typescript\"", "clients: [\"typescript\"]"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("converted YAML lacks %q:\n%s", want, data)
		}
	}
}

// TestProjectStorageAndVaultRenderAtTheTopLevel pins the corpus project form.
// Storage and vault are top-level YAML keys and the server block stays under
// generation; the first gate measurement failed the storage and vault tasks
// because the conversion wrote them where the compiler does not read them.
func TestProjectStorageAndVaultRenderAtTheTopLevel(t *testing.T) {
	source := "project {\n  name: \"files\"\n  server: \"typescript\"\n  clients: [\"typescript\"]\n  storage: { assets: \"s3\" }\n  vault: \"chickinfry\"\n}\n"
	file := Parse("chik/project.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"project:\n  name: \"files\"\n",
		"storage:\n  assets: \"s3\"\n",
		"generation:\n  server:\n    language: \"typescript\"\n",
		"vault: \"chickinfry\"\n",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("converted YAML lacks %q:\n%s", want, data)
		}
	}
}

// TestOperationParametersShareOneInputMapping pins the YAML input shape: one
// input mapping per operation no matter how many parameters it takes. Opening
// the mapping per parameter repeats the key, which YAML rejects.
func TestOperationParametersShareOneInputMapping(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      create(title: string, subtitle: string) -> { id: id }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "input:") != 1 {
		t.Fatalf("an operation with two parameters must open one input mapping:\n%s", data)
	}
	for _, want := range []string{"title: {type: string}", "subtitle: {type: string}"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("converted YAML lacks %q:\n%s", want, data)
		}
	}
}

// TestObjectListRendersItemListFields pins the corpus list form: the element
// record becomes the list's item fields. Reading the element without its
// list modifier used to render the record bare, which the compiler refuses
// with unknown field.
func TestObjectListRendersItemListFields(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      list() -> { rows: { id: id, title: string }[] }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	want := "rows: {type: list, items: {id: {type: id}, title: {type: string}}}"
	if !strings.Contains(string(data), want) {
		t.Fatalf("converted YAML lacks %q:\n%s", want, data)
	}
}

// TestNullableObjectListInputRenders keeps the nullable marker with the list
// it modifies instead of narrowing the field to a nullable scalar.
func TestNullableObjectListInputRenders(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      create(tags: { name: string }[]?) -> { id: id }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	want := "tags: {type: list, items: {name: {type: string}}, nullable: true}"
	if !strings.Contains(string(data), want) {
		t.Fatalf("converted YAML lacks %q:\n%s", want, data)
	}
}

// TestScalarListIsRefusedNotNarrowed keeps a list of scalars from compiling
// as the scalar alone: the YAML surface declares object lists only, so the
// refusal names the list the author wrote.
func TestScalarListIsRefusedNotNarrowed(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      list() -> { rows: string[] }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	_, err := ToYAML(file)
	if err == nil || !strings.Contains(err.Error(), "object lists only") {
		t.Fatalf("a list of scalars must be refused, not narrowed: %v", err)
	}
	if !strings.Contains(err.Error(), "chik/api.chik:") {
		t.Fatalf("error has no file and position:\n%v", err)
	}
}

// TestInlineRecordOutsideListItemIsRefused keeps a nested record output from
// being rendered as field metadata the compiler rejects as unknown fields:
// the YAML surface has no record kind, so the refusal says so where the
// record was written.
func TestInlineRecordOutsideListItemIsRefused(t *testing.T) {
	source := "services {\n  posts {\n    operations {\n      get(id: id) -> { post: { id: id } }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	_, err := ToYAML(file)
	if err == nil || !strings.Contains(err.Error(), "an inline record has no YAML form outside a list item; flatten its fields") {
		t.Fatalf("a bare record output must be refused with its reason: %v", err)
	}
}

// TestNamedRecordOutsideListItemIsRefusedAtTheUse keeps the refusal on the
// line the author can fix: the use names the type, so the message points
// there instead of at the declaration, and names what the surface accepts.
func TestNamedRecordOutsideListItemIsRefusedAtTheUse(t *testing.T) {
	source := "types {\n  Post = { id: id }\n}\nservices {\n  posts {\n    operations {\n      get() -> { post: Post }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	_, err := ToYAML(file)
	if err == nil || !strings.Contains(err.Error(), "Post has no YAML form outside a list item; use it as a list's item type") {
		t.Fatalf("a named record in field position must be refused at its use with the alternative: %v", err)
	}
	if !strings.Contains(err.Error(), "chik/api.chik:7:") {
		t.Fatalf("the refusal must point at the use, not the declaration:\n%v", err)
	}
}

// TestNamedRecordExpandsInListItem keeps a types block usable in the list
// position the YAML surface expresses: the named record expands into the list
// item fields.
func TestNamedRecordExpandsInListItem(t *testing.T) {
	source := "types {\n  Post = { id: id, title: string }\n}\nservices {\n  posts {\n    operations {\n      list() -> { rows: Post[] }\n    }\n  }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	want := "rows: {type: list, items: {id: {type: id}, title: {type: string}}}"
	if !strings.Contains(string(data), want) {
		t.Fatalf("converted YAML lacks %q:\n%s", want, data)
	}
}

// TestTypesOnlyFileRendersNothing pins the absent-role semantics of a file
// that only names records: nothing it declares reaches the YAML form, so the
// check treats the role as absent instead of handing the compiler an empty
// file it can only refuse with EOF.
func TestTypesOnlyFileRendersNothing(t *testing.T) {
	source := "types {\n  Post = { id: id }\n}\n"
	file := Parse("chik/api.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("a types-only file must render to nothing:\n%s", data)
	}
}

// TestCapabilitiesConvertToTheProjectKey keeps the .chik form the beta scope
// documents reaching the compiler. The retired extensions list is not an
// equivalent: it has no author handle, and the compiler no longer reads it.
func TestCapabilitiesConvertToTheProjectKey(t *testing.T) {
	source := "project {\n  name: \"acme\"\n  server: \"typescript\"\n  capabilities: { control_plane: \"platform/control-plane\" }\n}\n"
	file := Parse("chik/project.chik", source)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
	data, err := ToYAML(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "capabilities:\n  control_plane: \"platform/control-plane\"\n") {
		t.Fatalf("converted YAML lacks the capability map:\n%s", data)
	}
	if strings.Contains(string(data), "extensions:") {
		t.Fatalf("converted YAML uses the retired extensions key:\n%s", data)
	}
	// Formatting must not change what the file means.
	formatted := Format(file)
	again, err := ToYAML(Parse("chik/project.chik", formatted))
	if err != nil || string(again) != string(data) {
		t.Fatalf("format changed the converted YAML: %v\n%s", err, again)
	}
}

// TestDuplicateCapabilityPointsAtTheSource keeps a duplicate from reaching the
// YAML decoder, which would report a line the .chik file does not have.
func TestDuplicateCapabilityPointsAtTheSource(t *testing.T) {
	source := "project {\n  name: \"acme\"\n  capabilities: { control_plane: \"platform/control-plane\", control_plane: \"platform/control-plane\" }\n}\n"
	file := Parse("chik/project.chik", source)
	_, err := ToYAML(file)
	if err == nil {
		t.Fatal("a duplicate capability was accepted")
	}
	if !strings.Contains(err.Error(), "chik/project.chik:3:") {
		t.Fatalf("error does not point at the source line:\n%v", err)
	}
}

// TestCapabilitiesMustBeANamedMap keeps a list from being read as a map. The
// name is the author's handle, and a list has none.
func TestCapabilitiesMustBeANamedMap(t *testing.T) {
	source := "project {\n  name: \"acme\"\n  capabilities: [\"platform/control-plane\"]\n}\n"
	file := Parse("chik/project.chik", source)
	_, err := ToYAML(file)
	if err == nil || !strings.Contains(err.Error(), "name-keyed map") {
		t.Fatalf("a capability list was accepted: %v", err)
	}
}
