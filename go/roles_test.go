package chiklang

import "testing"

// TestRoleExamplesParse pins the role-file examples proposed for confirmation.
// Each is the shape the corpus and the beta scope describe; if the parser stops
// accepting one, the proposal and the parser have diverged.
func TestRoleExamplesParse(t *testing.T) {
	examples := map[string]string{
		"project": `project {
  name: "acme";
  server: "typescript";
  secrets: ["STRIPE_KEY", "MAIL_TOKEN"];
  clients: ["go", "typescript"];
  storage: { assets: "s3", uploads: "s3" };
  vault: "chickinfry";
}
`,
		"schema": `database {
  profile: "postgresql";
  tables {
    posts {
      columns {
        id: { type: id, default: { kind: uuid } };
        title: { type: string, default: { kind: string, literal: "Untitled" } };
        body: { type: string, nullable: true };
        author: { type: id, references: { auth: user } };
        published_at: { type: timestamp };
      }
      primary_key: ["id"];
      indexes {
        by_author { columns: ["author"] }
      }
      checks {
        title_present { column: title, non_blank: true }
      }
    }
  }
}
`,
		"auth": `auth {
  methods: ["email-password"];
  origins: ["https://app.example.test"];
  signup: "open";
  browser_session: false;
  session: {
    idle_timeout_seconds: 2592000,
    absolute_lifetime_seconds: 0,
  };
  email: {
    verification: "required",
    transport: "platform",
  };
  rate_limits: {
    signup: { signal: "email", max: 5, window_seconds: 60 },
  };
}
`,
		"clients": `clients {
  web: { template: "typescript", api: "full" };
  admin: { template: "react", api: "none" };
}
`,
		"project with capabilities": `project {
  name: "acme";
  server: "typescript";
  capabilities: { control_plane: "platform/control-plane" };
}
`,
	}
	for name, source := range examples {
		t.Run(name, func(t *testing.T) {
			file := Parse("chik/"+name+".chik", source)
			if len(file.Diagnostics) != 0 {
				t.Fatalf("diagnostics: %+v", file.Diagnostics)
			}
			if got := Serialize(file); got != source {
				t.Fatalf("roundtrip changed the source\ngot:\n%s", got)
			}
		})
	}
}

// TestNumbersAndInlineRecords covers the shapes the role examples rely on.
func TestNumbersAndInlineRecords(t *testing.T) {
	file := Parse("chik/auth.chik", "auth {\n  session: { idle_timeout_seconds: 2592000 };\n}\n")
	if len(file.Diagnostics) != 0 {
		t.Fatalf("diagnostics: %+v", file.Diagnostics)
	}
}
