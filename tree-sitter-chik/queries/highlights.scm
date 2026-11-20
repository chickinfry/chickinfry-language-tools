; Highlight queries for the .chik grammar. Editors that read tree-sitter
; queries (Neovim, Helix, Zed) get the same scopes as the TextMate grammar.
;
; Only standard captures are used. tree-sitter highlight --check names any
; other capture, and scripts/check-highlights.mjs fails on that output in CI.

(line_comment) @comment
(block_comment) @comment

(string) @string
(number) @number
(boolean) @constant.builtin

[
  "types"
  "services"
  "routes"
  "operations"
  "union"
  "public"
  "at"
  "project"
  "database"
  "auth"
  "clients"
  "map"
  "Page"
] @keyword

[
  "id"
  "string"
  "timestamp"
  "json"
] @type.builtin

(method) @constant

(type_declaration name: (identifier) @type)
(variant name: (identifier) @constant)
(service name: (identifier) @type)

(operation name: (identifier) @function)
(route name: (identifier) @function)

(parameter name: (identifier) @variable.parameter)
(field name: (identifier) @property)
(config_entry name: (identifier) @property)
(config_section name: (identifier) @property)

[
  "{"
  "}"
  "("
  ")"
  "["
  "]"
  "<"
  ">"
] @punctuation.bracket

[
  ":"
  ","
  ";"
  "="
  "->"
  "?"
] @punctuation.delimiter

