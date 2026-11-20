# Beta scope

The beta ships `.chik` for exactly what YAML expresses today. No new
declaration feature arrives with the language. Routes, custom operation
addresses, and `Page<T>` stay in the later release that adds those features to
YAML as well.

This page lists what the beta grammar must cover, derived from the current YAML
schema. Anything not listed here is out of scope, even though the grammar draft
describes it.

## project.chik

```text
project {
  name: "demo";
  server: "typescript";
  clients: ["go", "typescript"];
  secrets: ["API_KEY"];
  storage: { assets: "s3" };
  vault: "chickinfry";
}
```

| YAML | `.chik` |
| --- | --- |
| `project.name` | `project { name: "..." }` |
| `generation.server.language` | `server: "..."` |
| `generation.server.secrets` | `secrets: ["..."]` |
| `generation.clients` | `clients: ["..."]` |
| `storage.<name>` | `storage: { name: "s3" }` |
| `vault` | `vault: "chickinfry"` |

`realtime` is not in the beta: it is rejected by the current release build and
is not part of the YAML beta surface.

## schema.chik

```text
database {
  profile: "postgresql";
  tables {
    posts {
      columns {
        id: { type: id, primary_key: true, default: { kind: uuid } };
        title: { type: string, default: { kind: string, literal: "Untitled" } };
        author: { type: id, references: { auth: user } };
        note: { type: string, nullable: true };
      }
      primary_key: ["id"];
      unique { by_email { columns: ["email"] } }
      foreign_keys {
        author { columns: ["author"], references: { table: "users", columns: ["id"] } }
      }
      indexes {
        recent { columns: ["created_at"], unique: false }
      }
      checks {
        title_present { column: title, non_blank: true }
      }
    }
  }
}
```

Every YAML column key maps one to one: `type`, `primary_key`, `nullable`,
`unique`, `references`, `default`. Table keys: `columns`, `primary_key`,
`unique`, `foreign_keys`, `indexes`, `checks`.

A check predicate keeps the typed tree shape from YAML: `column`, `all`, `any`,
`not`, `when`, `then`, `in`, `eq`, `ne`, `lt`, `lte`, `gt`, `gte`,
`is_null`, `non_blank`.

## api.chik

```text
services {
  posts {
    operations {
      get(id: id) -> { post: { id: id, title: string } };
      list() -> {
        rows: { id: id, title: string }[];
      };
      create(title: string, tags: string[]?) -> { id: id };
    }
  }
}
```

The beta operation form has no `public` selector and no `at` address: both are
later-release features. Input and output fields keep the YAML metadata
`optional`, `nullable`, `default`, and list `items`, plus the `one_of` group
map.

```text
create(title: string, tag: { type: string, optional: true })
  -> { id: id };
```

## clients.chik

```text
clients {
  web: { template: "typescript", api: "full" };
}
```

## auth.chik

```text
auth {
  methods: ["email-password"];
  origins: ["https://app.example.test"];
  signup: "open";
  browser_session: false;
  session: { idle_timeout_seconds: 2592000 };
  email: { verification: "required", transport: "platform" };
  rate_limits: { signup: { signal: "email", max: 5, window_seconds: 60 } };
}
```

## extensions.chik

```text
extensions {
  platform_control_plane: "platform/control-plane";
}
```

## What the beta grammar does not include

| Feature | Why it waits |
| --- | --- |
| `routes` block | New capability, not in YAML today. |
| `at METHOD "path"` | New capability. |
| `public` selector | The YAML form exists but the beta grammar keeps the current YAML key. |
| `Page<T>` | New capability. |
| `union(kind)` | New capability. |
| `json` | New capability. |
| `map<string, T>` | New capability. |
| Named types in `types` | New capability. |

Named types, unions, and `Page` appear in the grammar draft because they are
part of the language's direction. They are not beta requirements, and a beta
implementation may reject them with a clear message rather than accept them
silently.

## Consequence for the corpus

Every case in `conformance/` must fall inside this scope. A case that exercises
a later-release feature belongs to that release, not to beta.
