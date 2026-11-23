package chiklang

import (
	"strings"
	"testing"
)

// TestConvertKeepsComments proves a comment written in YAML reaches the .chik
// output. A declaration carries the reasoning for a field, and a conversion
// that drops it makes the converted file poorer than the one it replaced.
// Every role is covered: comment preservation in one role only would make the
// guarantee depend on which file the reader happened to convert.
func TestConvertKeepsComments(t *testing.T) {
	cases := []struct{ role, source, want, alsoWant string }{
		{"api", "# The blog service.\nservices:\n  blog:\n    # One page of posts.\n    operations:\n      list_posts:\n        output:\n          id: {type: id}\n", "// The blog service.", "// One page of posts."},
		{"schema", "# The records table.\ndatabase:\n  profile: postgresql\n  tables:\n    records:\n      columns:\n        # stable id\n        id: {type: id}\n", "// The records table.", "// stable id"},
		{"auth", "# Login policy.\nauth:\n  methods: [email-password]\n  # how long a session lives\n  session: { idle_timeout_seconds: 2592000 }\n", "// Login policy.", "// how long a session lives"},
		{"clients", "# Ship the web app.\nclients:\n  web: { template: typescript, api: full }\n", "// Ship the web app.", ""},
		{"project", "# Acme.\nproject:\n  name: acme\n# realtime chat\nrealtime: chickinfry\n", "// Acme.", "// realtime chat"},
	}
	for _, test := range cases {
		t.Run(test.role, func(t *testing.T) {
			converted, diagnostics := Convert(test.role, []byte(test.source))
			if len(diagnostics) != 0 {
				t.Fatalf("refused: %+v", diagnostics)
			}
			for _, want := range []string{test.want, test.alsoWant} {
				if want != "" && !strings.Contains(converted, want) {
					t.Fatalf("comment %q lost:\n%s", want, converted)
				}
			}
			// The converted file must still parse, comments included.
			if file := Parse("chik/"+test.role+".chik", converted); len(file.Diagnostics) != 0 {
				t.Fatalf("converted output does not parse: %+v\n%s", file.Diagnostics, converted)
			}
		})
	}
}

// TestConvertCommentRoundtripsThroughFormat keeps conversion and formatting
// agreeing on comments: formatting a converted file must change nothing.
func TestConvertCommentRoundtripsThroughFormat(t *testing.T) {
	source := "services:\n  blog:\n    # A note.\n    operations:\n      x:\n        output:\n          id: {type: id}\n"
	converted, diagnostics := Convert("api", []byte(source))
	if len(diagnostics) != 0 {
		t.Fatalf("refused: %+v", diagnostics)
	}
	if formatted := Format(Parse("chik/api.chik", converted)); formatted != converted {
		t.Fatalf("conversion is not canonical with comments\nconverted:\n%s\nformatted:\n%s", converted, formatted)
	}
}
