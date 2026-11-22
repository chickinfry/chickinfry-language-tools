package chiklang

import "testing"

// TestFormattedBlockBodyHasNoTrailingComma covers a defect where the formatter
// added a comma before a closing brace in a block body, so a converted auth
// declaration could not be parsed back. A record takes the comma; a block body
// ends with a line break.
func TestFormattedBlockBodyHasNoTrailingComma(t *testing.T) {
	source := "auth {\n  methods: [\"email-password\"]\n  browser_session: false\n}\n"
	got := Format(Parse("chik/auth.chik", source))
	if got != source {
		t.Fatalf("formatted =\n%s\nwant:\n%s", got, source)
	}
	if file := Parse("chik/auth.chik", got); len(file.Diagnostics) != 0 {
		t.Fatalf("formatted output does not parse: %+v", file.Diagnostics)
	}
}

// TestFormattedRecordKeepsTrailingComma keeps the other half of the rule: a
// record that spans lines keeps a comma on its last entry.
func TestFormattedRecordKeepsTrailingComma(t *testing.T) {
	source := "auth {\n  session: {\n    idle_timeout_seconds: 2592000\n  }\n}\n"
	want := "auth {\n  session: {\n    idle_timeout_seconds: 2592000,\n  }\n}\n"
	if got := Format(Parse("chik/auth.chik", source)); got != want {
		t.Fatalf("formatted =\n%s\nwant:\n%s", got, want)
	}
}

// TestFormattedMultiLineListKeepsTrailingComma covers the defect that narrowed
// the comma to records: a list or a parameter list that spans lines takes a
// trailing comma on its last entry, as the canonical form requires.
func TestFormattedMultiLineListKeepsTrailingComma(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{
			"list",
			"auth {\n  methods: [\n    \"email-password\"\n  ]\n}\n",
			"auth {\n  methods: [\n    \"email-password\",\n  ]\n}\n",
		},
		{
			"parameters",
			"services {\n  s {\n    operations {\n      o(\n        a: string\n      ) -> {}\n    }\n  }\n}\n",
			"services {\n  s {\n    operations {\n      o(\n        a: string,\n      ) -> {}\n    }\n  }\n}\n",
		},
		{
			"list beside a block entry",
			"auth {\n  methods: [\n    \"email-password\"\n  ]\n  browser_session: false\n}\n",
			"auth {\n  methods: [\n    \"email-password\",\n  ]\n  browser_session: false\n}\n",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := Format(Parse("chik/x.chik", test.source))
			if got != test.want {
				t.Fatalf("formatted =\n%s\nwant:\n%s", got, test.want)
			}
			if file := Parse("chik/x.chik", got); len(file.Diagnostics) != 0 {
				t.Fatalf("formatted output does not parse: %+v", file.Diagnostics)
			}
			if again := Format(Parse("chik/x.chik", got)); again != got {
				t.Fatalf("not idempotent:\n%s", again)
			}
		})
	}
}

// TestFormattedEmptyMultiLineListGainsNoComma keeps an empty list or parameter
// list that spans lines parseable: there is no last entry to take a comma.
func TestFormattedEmptyMultiLineListGainsNoComma(t *testing.T) {
	cases := []struct{ name, source string }{
		{"list", "auth {\n  methods: [\n  ]\n}\n"},
		{"parameters", "services {\n  s {\n    operations {\n      o(\n      ) -> {}\n    }\n  }\n}\n"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if file := Parse("chik/x.chik", test.source); len(file.Diagnostics) != 0 {
				t.Fatalf("input does not parse: %+v", file.Diagnostics)
			}
			got := Format(Parse("chik/x.chik", test.source))
			if got != test.source {
				t.Fatalf("formatted =\n%s\nwant:\n%s", got, test.source)
			}
			if file := Parse("chik/x.chik", got); len(file.Diagnostics) != 0 {
				t.Fatalf("formatted output does not parse: %+v", file.Diagnostics)
			}
		})
	}
}

// TestFormattedNamedAndReturnRecordsKeepTrailingComma covers the records the
// brace-after-colon rule missed: a named type after an equals sign and an
// operation result after an arrow are records, so a multi-line one takes a
// trailing comma on its last entry.
func TestFormattedNamedAndReturnRecordsKeepTrailingComma(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{
			"named type",
			"types {\n  Post = {\n    id: id,\n    title: string\n  }\n}\n",
			"types {\n  Post = {\n    id: id,\n    title: string,\n  }\n}\n",
		},
		{
			"operation result",
			"services {\n  s {\n    operations {\n      o() -> {\n        id: id,\n        title: string\n      }\n    }\n  }\n}\n",
			"services {\n  s {\n    operations {\n      o() -> {\n        id: id,\n        title: string,\n      }\n    }\n  }\n}\n",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if file := Parse("chik/x.chik", test.source); len(file.Diagnostics) != 0 {
				t.Fatalf("input does not parse: %+v", file.Diagnostics)
			}
			got := Format(Parse("chik/x.chik", test.source))
			if got != test.want {
				t.Fatalf("formatted =\n%s\nwant:\n%s", got, test.want)
			}
			if file := Parse("chik/x.chik", got); len(file.Diagnostics) != 0 {
				t.Fatalf("formatted output does not parse: %+v", file.Diagnostics)
			}
			if again := Format(Parse("chik/x.chik", got)); again != got {
				t.Fatalf("not idempotent:\n%s", again)
			}
		})
	}
}
