# Error codes

A diagnostic carries one of these codes. A code is stable: implementations may
reword the message, and may add detail, but they do not rename or reuse a code
for a different problem.

Codes are grouped by the stage that produces them.

## Lexical

| Code | Meaning |
| --- | --- |
| `chik_unexpected_character` | A character that cannot start a token. |
| `chik_unterminated_string` | A string literal ends at end of line or file. |
| `chik_invalid_escape` | A backslash escape that is not a JSON escape. |
| `chik_unterminated_comment` | A block comment has no closing delimiter. |

## Syntax

| Code | Meaning |
| --- | --- |
| `chik_expected_token` | A required token is missing or different. |
| `chik_expected_type` | A type expression was required. |
| `chik_expected_name` | An identifier was required. |
| `chik_duplicate_block` | `types`, `services`, or `routes` appears twice. |
| `chik_empty_file` | The file declares no block. |

## Names

| Code | Meaning |
| --- | --- |
| `chik_invalid_name` | An identifier is not the case its position requires. |
| `chik_duplicate_name` | Two declarations share a name in one scope. |
| `chik_reserved_name` | A name the generated code reserves. |

## Types

| Code | Meaning |
| --- | --- |
| `chik_unknown_type` | A named type is not declared. |
| `chik_cyclic_type` | A named type refers to itself directly or through others. |
| `chik_invalid_modifier` | `?` or `[]` applied where it is not allowed. |
| `chik_invalid_map_key` | A map key that is not `string`. |
| `chik_page_not_output` | `Page<T>` used outside an operation output. |
| `chik_union_variant_count` | A union with fewer than two variants. |
| `chik_union_reserved_field` | A variant redeclares the discriminator field. |
| `chik_empty_record` | An empty record where at least one field is required. |

## Operations and routes

| Code | Meaning |
| --- | --- |
| `chik_invalid_method` | A method outside the allowed set. |
| `chik_invalid_path` | A path that breaks the address rules. |
| `chik_path_parameter` | A path placeholder that does not match an input. |
| `chik_route_overlap` | Two addresses that would match the same request. |
| `chik_reserved_service` | A service name the platform owns. |

## Conversion

| Code | Meaning |
| --- | --- |
| `chik_convert_unsupported` | A YAML declaration with no `.chik` form. |
| `chik_convert_ambiguous` | A YAML declaration that maps to more than one form. |

## Not in this corpus

Warnings are not part of the corpus yet. A declaration that produces only
warnings is formatted and converted normally.
