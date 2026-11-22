package chiklang

import (
	"strings"
	"testing"
)

// TestConvertRefusesLoss covers the constructs with no .chik form. Dropping
// one would leave a converted file that looks complete but is not the same
// declaration.
func TestConvertRefusesLoss(t *testing.T) {
	cases := []struct{ name, role, source, want string }{
		{"one_of group", "api", "services:\n  s:\n    operations:\n      o:\n        input:\n          rows:\n            type: list\n            optional: true\n            items:\n              a: {type: string, optional: true}\n              b: {type: string, optional: true}\n            one_of: {x: [a, b]}\n", "one_of"},
		{"input default", "api", "services:\n  s:\n    operations:\n      o:\n        input:\n          a: {type: string, optional: true, default: \"x\"}\n", "default"},
		{"public selector", "api", "services:\n  s:\n    operations:\n      o:\n        public: true\n", "public"},
		{"operation method", "api", "services:\n  s:\n    operations:\n      o:\n        method: GET\n", "method"},
		{"not a mapping", "project", "- one\n", "not a mapping"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			output, diagnostics := Convert(test.role, []byte(test.source))
			if len(diagnostics) == 0 {
				t.Fatalf("converted without refusing:\n%s", output)
			}
			if output != "" {
				t.Fatalf("a refusal must not return a partial result: %q", output)
			}
			if diagnostics[0].Code != CodeConvertUnsupported {
				t.Fatalf("code = %q, want %q", diagnostics[0].Code, CodeConvertUnsupported)
			}
			if !strings.Contains(diagnostics[0].Message, test.want) {
				t.Fatalf("message = %q, want it to mention %q", diagnostics[0].Message, test.want)
			}
		})
	}
}

// TestConvertKeepsColumnDefault keeps the other half: a database column
// default has a form, so it must convert rather than be refused.
func TestConvertKeepsColumnDefault(t *testing.T) {
	source := "database:\n  profile: postgresql\n  tables:\n    t:\n      columns:\n        id: {type: id, primary_key: true, default: {kind: uuid}}\n"
	output, diagnostics := Convert("schema", []byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("refused: %+v", diagnostics)
	}
	if !strings.Contains(output, "default: { kind: uuid }") {
		t.Fatalf("default lost:\n%s", output)
	}
}
