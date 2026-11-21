// Grammar for .chik declaration files. The contract is spec/grammar.md and
// spec/beta-scope.md; the conformance corpus decides the bytes.
//
// This grammar exists for editor highlighting, so it is deliberately
// tolerant. Semantic rules (name case, unknown types, duplicate blocks,
// address overlap) are reported by the Go parser, not here. Comments and
// whitespace are extras: the CST with trivia lives in the Go implementation.
module.exports = grammar({
  name: "chik",

  extras: $ => [/\s/, $.line_comment, $.block_comment],

  word: $ => $.identifier,

  rules: {
    source_file: $ => repeat($._block),

    _block: $ => choice(
      $.types_block,
      $.services_block,
      $.routes_block,
      $.config_block,
    ),

    line_comment: _ => token(seq("//", /.*/)),
    block_comment: _ => token(seq("/*", /[^*]*\*+([^/*][^*]*\*+)*/, "/")),

    types_block: $ => seq(
      "types",
      "{",
      repeat(seq($.type_declaration, optional(";"))),
      "}",
    ),

    type_declaration: $ => seq(
      field("name", $.identifier),
      "=",
      choice($.record, $.union),
    ),

    union: $ => seq(
      "union",
      "(",
      field("tag", $.identifier),
      ")",
      "{",
      repeat($.variant),
      "}",
    ),

    variant: $ => seq(
      field("name", $.identifier),
      ":",
      $.record,
      optional(";"),
    ),

    record: $ => seq("{", optional($.field_list), "}"),

    field_list: $ => seq(
      $.field,
      // Commas are required between fields: the grammar's field list is
      // `field, {",", field}, [","]`. The Go-style rule that a line break
      // ends an entry applies to block entries such as `name: "demo"`, not
      // to the fields inside a record.
      repeat(seq(",", $.field)),
      optional(","),
    ),

    field: $ => seq(
      field("name", $.identifier),
      ":",
      field("type", $._type),
      optional(field("optional", $.optional_modifier)),
    ),

    _type: $ => choice(
      $.type_list,
      $.type_nullable,
      $.record,
      $.map_type,
      $.page_type,
      $.primitive_type,
      $.identifier,
    ),

    type_list: $ => prec.left(seq(field("element", $._type), "[]")),
    type_nullable: $ => prec.left(seq(field("inner", $._type), "?")),

    primitive_type: _ => choice("id", "string", "timestamp", "json"),

    map_type: $ => seq("map", "<", "string", ",", field("value", $._type), ">"),
    page_type: $ => seq("Page", "<", field("value", $._type), ">"),

    services_block: $ => seq(
      "services",
      "{",
      repeat($.service),
      "}",
    ),

    service: $ => seq(
      field("name", $.identifier),
      "{",
      "operations",
      "{",
      repeat(seq($.operation, optional(";"))),
      "}",
      "}",
    ),

    operation: $ => seq(
      optional("public"),
      field("name", $.identifier),
      "(",
      optional($.parameter_list),
      ")",
      "->",
      $.record,
      optional($.address),
    ),

    parameter_list: $ => seq(
      $.parameter,
      repeat(seq(",", $.parameter)),
      optional(","),
    ),

    parameter: $ => seq(
      field("name", $.identifier),
      ":",
      field("type", $._type),
      // `optional` is a word after the type, not a reserved keyword, so a
      // field named optional still parses. It marks an input the caller may
      // omit, which is a different thing from the nullable `?` modifier.
      optional(field("optional", $.optional_modifier)),
    ),

    // A named node so an editor can highlight the marker and a test can prove
    // the word was consumed rather than left trailing.
    optional_modifier: _ => "optional",

    address: $ => seq(
      "at",
      field("method", $.method),
      field("path", $.string),
    ),

    method: _ => token(choice(
      "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS",
    )),

    routes_block: $ => seq(
      "routes",
      "{",
      repeat($.route),
      "}",
    ),

    route: $ => seq(
      field("name", $.identifier),
      field("method", $.method),
      field("path", $.string),
      optional(";"),
    ),

    // project.chik, schema.chik, auth.chik and clients.chik share one shape:
    // named sections holding key/value entries. Which keys each role allows
    // is a schema question the Go parser answers.
    //
    // extensions.chik is not here on purpose. The roles decision retired it
    // and moved its one entry into project.chik under `capabilities`, so a
    // file that still starts with that keyword must not highlight as valid.
    config_block: $ => seq(
      field("role", $.block_role),
      "{",
      repeat($._config_item),
      "}",
    ),

    block_role: _ => choice(
      "project", "database", "auth", "clients",
    ),

    _config_item: $ => choice(
      $.config_entry,
      $.config_section,
    ),

    config_entry: $ => seq(
      field("name", $.identifier),
      ":",
      field("value", $.config_value),
      optional($._separator),
    ),

    config_section: $ => seq(
      field("name", $.identifier),
      "{",
      repeat($._config_item),
      "}",
      optional($._separator),
    ),

    config_value: $ => choice(
      $.string,
      $.number,
      $.boolean,
      $.config_record,
      $.config_list,
      $.identifier,
    ),

    config_record: $ => seq("{", repeat($._config_item), "}"),

    config_list: $ => seq(
      "[",
      repeat(choice($.config_value, $._separator)),
      "]",
    ),

    _separator: _ => choice(";", ","),

    boolean: _ => choice("true", "false"),
    number: _ => token(seq(optional("-"), /[0-9]+/, optional(seq(".", /[0-9]+/)))),

    string: _ => token(seq(
      '"',
      repeat(choice(
        /[^"\\\n]/,
        /\\(["\\/bfnrt]|u[0-9a-fA-F]{4})/,
      )),
      '"',
    )),

    identifier: _ => /[A-Za-z_][A-Za-z0-9_]*/,
  },
});
