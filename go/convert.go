package chiklang

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Convert turns one YAML declaration into the .chik source that expresses the
// same declaration. It refuses, with CHK0501, anything it cannot express
// without loss rather than guessing.
//
// The input is the YAML document for a single role file, so the caller decides
// which file it read.
func Convert(role string, data []byte) (string, []Diagnostic) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: err.Error()}}
	}
	if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "the document is not a mapping"}}
	}
	mapping := root.Content[0]
	if role == "extensions" {
		return "", []Diagnostic{{
			Code: CodeConvertUnsupported, Severity: "error",
			Message: "extensions.yml is retired; use ConvertExtensions to move it into project.chik",
		}}
	}
	if refusals := checkConvertible(role, mapping); len(refusals) != 0 {
		diagnostics := make([]Diagnostic, 0, len(refusals))
		for _, r := range refusals {
			diagnostics = append(diagnostics, Diagnostic{
				Code: CodeConvertUnsupported, Severity: "error", Path: r.path, Message: r.message,
			})
		}
		return "", diagnostics
	}
	// The converter writes the grammar's structure; the formatter then applies
	// the canonical layout, so a converted file is already what chik fmt would
	// produce and never carries a semicolon the formatter would drop.
	text, diagnostics := convertRole(role, mapping)
	if len(diagnostics) != 0 {
		return text, diagnostics
	}
	// A comment above the first key is attached to the document node, not to
	// the mapping, so it is carried over separately.
	if head := commentLines(&root, ""); head != "" {
		text = head + text
	}
	return Format(Parse("chik/"+role+".chik", text)), nil
}

func convertRole(role string, mapping *yaml.Node) (string, []Diagnostic) {
	switch role {
	case "project":
		return convertProject(mapping)
	case "api":
		return convertAPI(mapping)
	case "schema":
		return convertSchema(mapping)
	case "auth":
		return convertAuth(mapping)
	case "clients":
		return convertClients(mapping)
	}
	return "", []Diagnostic{{
		Code: CodeConvertUnsupported, Severity: "error",
		Message: fmt.Sprintf("converting %s is not implemented yet", role),
	}}
}

// pair is one key and value in document order.
type pair struct {
	name string
	node *yaml.Node
	// key is the mapping key, where YAML attaches a comment written above
	// the entry.
	key *yaml.Node
}

// pairs lists a mapping's entries in the order the author wrote them, which is
// the order the converted file keeps.
func pairs(node *yaml.Node) []pair {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	result := make([]pair, 0, len(node.Content)/2)
	for index := 0; index+1 < len(node.Content); index += 2 {
		result = append(result, pair{name: node.Content[index].Value, node: node.Content[index+1], key: node.Content[index]})
	}
	return result
}

func fields(node *yaml.Node) map[string]*yaml.Node {
	result := map[string]*yaml.Node{}
	for _, entry := range pairs(node) {
		result[entry.name] = entry.node
	}
	return result
}

func scalar(node *yaml.Node) (string, bool) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag == "!!null" {
		return "", false
	}
	return node.Value, true
}

// literal renders a scalar the way .chik writes it: a string is quoted, a
// boolean and a number stay bare, and a null is absent.
func literal(node *yaml.Node) (string, bool) {
	value, ok := scalar(node)
	if !ok {
		return "", false
	}
	switch node.Tag {
	case "!!bool", "!!int", "!!float":
		return value, true
	}
	return fmt.Sprintf("%q", value), true
}

func convertProject(mapping *yaml.Node) (string, []Diagnostic) {
	var out strings.Builder
	top := fields(mapping)
	out.WriteString(commentFor(mapping, "project", ""))
	out.WriteString("project {\n")
	if project := fields(top["project"]); project != nil {
		out.WriteString(commentFor(top["project"], "name", "  "))
		if name, ok := scalar(project["name"]); ok {
			fmt.Fprintf(&out, "  name: %q\n", name)
		}
	}
	out.WriteString(commentFor(mapping, "generation", "  "))
	if generation := fields(top["generation"]); generation != nil {
		out.WriteString(commentFor(top["generation"], "server", "  "))
		if server := fields(generation["server"]); server != nil {
			out.WriteString(commentFor(generation["server"], "language", "    "))
			if language, ok := scalar(server["language"]); ok {
				fmt.Fprintf(&out, "  server: %q\n", language)
			}
			out.WriteString(commentFor(generation["server"], "secrets", "  "))
			if secrets := server["secrets"]; secrets != nil && secrets.Kind == yaml.SequenceNode {
				fmt.Fprintf(&out, "  secrets: %s\n", quoteList(secrets))
			}
		}
		out.WriteString(commentFor(top["generation"], "clients", "  "))
		if clients := generation["clients"]; clients != nil && clients.Kind == yaml.SequenceNode {
			fmt.Fprintf(&out, "  clients: %s\n", quoteList(clients))
		}
	}
	out.WriteString(commentFor(mapping, "storage", "  "))
	if storage := top["storage"]; storage != nil && storage.Kind == yaml.MappingNode {
		var names []string
		for _, entry := range pairs(storage) {
			names = append(names, fmt.Sprintf("%s: %q", entry.name, entry.node.Value))
		}
		if len(names) != 0 {
			fmt.Fprintf(&out, "  storage: { %s }\n", strings.Join(names, ", "))
		}
	}
	out.WriteString(commentFor(mapping, "vault", "  "))
	if vault, ok := scalar(top["vault"]); ok && vault != "" {
		fmt.Fprintf(&out, "  vault: %q\n", vault)
	}
	// realtime is a project key in .chik. In YAML it sits beside project, so a
	// reader comparing the two files sees the same setting under the same name.
	out.WriteString(commentFor(mapping, "realtime", "  "))
	if realtime, ok := scalar(top["realtime"]); ok && realtime != "" {
		fmt.Fprintf(&out, "  realtime: %q\n", realtime)
	}
	out.WriteString("}\n")
	return out.String(), nil
}

func quoteList(node *yaml.Node) string {
	var values []string
	for _, item := range node.Content {
		values = append(values, fmt.Sprintf("%q", item.Value))
	}
	return "[" + strings.Join(values, ", ") + "]"
}

// convertSchema renders a database declaration. Tables, columns, keys, checks,
// and indexes keep the YAML key names so a reader can compare the two forms.
func convertSchema(mapping *yaml.Node) (string, []Diagnostic) {
	database := fields(mapping)["database"]
	if database == nil {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "database is missing"}}
	}
	top := fields(database)
	var out strings.Builder
	out.WriteString(commentFor(mapping, "database", ""))
	out.WriteString("database {\n")
	out.WriteString(commentFor(database, "profile", "  "))
	if profile, ok := scalar(top["profile"]); ok {
		fmt.Fprintf(&out, "  profile: %q\n", profile)
	}
	out.WriteString(commentFor(database, "tables", "  "))
	if tables := top["tables"]; tables != nil && tables.Kind == yaml.MappingNode {
		out.WriteString("  tables {\n")
		for _, table := range pairs(tables) {
			out.WriteString(commentLines(table.key, "    "))
			fmt.Fprintf(&out, "    %s {\n", table.name)
			body := fields(table.node)
			out.WriteString(commentFor(table.node, "columns", "      "))
			if columns := body["columns"]; columns != nil {
				out.WriteString("      columns {\n")
				for _, column := range pairs(columns) {
					out.WriteString(commentLines(column.key, "        "))
					fmt.Fprintf(&out, "        %s: %s\n", column.name, convertColumn(column.node))
				}
				out.WriteString("      }\n")
			}
			out.WriteString(commentFor(table.node, "primary_key", "      "))
			if key := body["primary_key"]; key != nil {
				fmt.Fprintf(&out, "      primary_key: %s\n", quoteList(key))
			}
			for _, group := range []string{"unique", "foreign_keys", "indexes", "checks"} {
				node := body[group]
				if node == nil {
					continue
				}
				out.WriteString(commentFor(table.node, group, "      "))
				fmt.Fprintf(&out, "      %s {\n", group)
				for _, entry := range pairs(node) {
					out.WriteString(commentLines(entry.key, "        "))
					fmt.Fprintf(&out, "        %s { %s }\n", entry.name, inlineMapping(entry.node))
				}
				out.WriteString("      }\n")
			}
			out.WriteString("    }\n")
		}
		out.WriteString("  }\n")
	}
	out.WriteString("}\n")
	return out.String(), nil
}

// convertColumn renders one column declaration. Keys keep their YAML spelling.
func convertColumn(node *yaml.Node) string {
	var parts []string
	for _, entry := range pairs(node) {
		switch entry.name {
		case "type":
			// A type is an identifier, not a string: `type: id`, not `type: "id"`.
			if value, ok := scalar(entry.node); ok {
				parts = append(parts, "type: "+value)
			}
		case "primary_key", "nullable", "unique":
			if value, ok := literal(entry.node); ok {
				parts = append(parts, entry.name+": "+value)
			}
		case "references":
			if auth := fields(entry.node)["auth"]; auth != nil {
				parts = append(parts, "references: { auth: user }")
			} else if value, ok := literal(entry.node); ok {
				parts = append(parts, "references: "+value)
			}
		case "default":
			kind, _ := scalar(fields(entry.node)["kind"])
			inner := "kind: " + kind
			if value, ok := literal(fields(entry.node)["literal"]); ok {
				inner += ", literal: " + value
			}
			parts = append(parts, "default: { "+inner+" }")
		}
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// inlineMapping renders a mapping on one line, which is the form the corpus
// uses for a key, index, or check body.
func inlineMapping(node *yaml.Node) string {
	var parts []string
	for _, entry := range pairs(node) {
		switch {
		case entry.node.Kind == yaml.SequenceNode:
			parts = append(parts, entry.name+": "+quoteList(entry.node))
		case entry.node.Kind == yaml.MappingNode:
			parts = append(parts, entry.name+": { "+inlineMapping(entry.node)+" }")
		default:
			if value, ok := literal(entry.node); ok {
				parts = append(parts, entry.name+": "+value)
			}
		}
	}
	return strings.Join(parts, ", ")
}

// convertAuth renders the authentication policy. The nested policy objects are
// written as inline records because the corpus keeps them on one line.
func convertAuth(mapping *yaml.Node) (string, []Diagnostic) {
	auth := fields(mapping)["auth"]
	if auth == nil {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "auth is missing"}}
	}
	var out strings.Builder
	out.WriteString(commentFor(mapping, "auth", ""))
	out.WriteString("auth {\n")
	for _, entry := range pairs(auth) {
		out.WriteString(commentLines(entry.key, "  "))
		switch entry.name {
		case "methods", "origins":
			fmt.Fprintf(&out, "  %s: %s\n", entry.name, quoteList(entry.node))
		case "session", "email", "rate_limits":
			out.WriteString("  " + entry.name + ": {\n")
			for _, inner := range pairs(entry.node) {
				out.WriteString(commentLines(inner.key, "    "))
				fmt.Fprintf(&out, "    %s,\n", inlinePair(inner))
			}
			out.WriteString("  }\n")
		default:
			if value, ok := literal(entry.node); ok {
				fmt.Fprintf(&out, "  %s: %s\n", entry.name, value)
			}
		}
	}
	out.WriteString("}\n")
	return out.String(), nil
}

func inlinePair(entry pair) string {
	switch {
	case entry.node.Kind == yaml.MappingNode:
		return entry.name + ": { " + inlineMapping(entry.node) + " }"
	case entry.node.Kind == yaml.SequenceNode:
		return entry.name + ": " + quoteList(entry.node)
	}
	if value, ok := literal(entry.node); ok {
		return entry.name + ": " + value
	}
	return entry.name
}

// convertClients renders named client apps.
func convertClients(mapping *yaml.Node) (string, []Diagnostic) {
	clients := fields(mapping)["clients"]
	if clients == nil {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "clients is missing"}}
	}
	var out strings.Builder
	out.WriteString(commentFor(mapping, "clients", ""))
	out.WriteString("clients {\n")
	for _, entry := range pairs(clients) {
		out.WriteString(commentLines(entry.key, "  "))
		fmt.Fprintf(&out, "  %s: { %s }\n", entry.name, inlineMapping(entry.node))
	}
	out.WriteString("}\n")
	return out.String(), nil
}

// convertExtensions renders the extension list as a named map, which is the
// form the role example proposes.
// ConvertExtensions moves an extensions.yml selection into the capabilities key
// of project.chik. It returns a fragment to merge into the project declaration.
func ConvertExtensions(data []byte) (string, []Diagnostic) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: err.Error()}}
	}
	if len(root.Content) != 1 || root.Content[0].Kind != yaml.MappingNode {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "the document is not a mapping"}}
	}
	return convertExtensions(root.Content[0])
}

func convertExtensions(mapping *yaml.Node) (string, []Diagnostic) {
	extensions := fields(mapping)["extensions"]
	if extensions == nil || extensions.Kind != yaml.SequenceNode {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "extensions is missing"}}
	}
	var entries []string
	for _, item := range extensions.Content {
		id, _ := scalar(fields(item)["id"])
		name := capabilityName(id)
		entries = append(entries, fmt.Sprintf("%s: %q", name, id))
	}
	return "  capabilities: { " + strings.Join(entries, ", ") + " }\n", nil
}

// capabilityName is the author-facing key for a capability identifier.
func capabilityName(id string) string {
	switch id {
	case "platform/control-plane":
		return "control_plane"
	}
	return strings.NewReplacer("/", "_", "-", "_").Replace(id)
}

func convertAPI(mapping *yaml.Node) (string, []Diagnostic) {
	services := fields(mapping)["services"]
	if services == nil || services.Kind != yaml.MappingNode {
		return "", []Diagnostic{{Code: CodeConvertUnsupported, Severity: "error", Message: "services is missing"}}
	}
	var out strings.Builder
	out.WriteString(commentFor(mapping, "services", ""))
	out.WriteString("services {\n")
	for _, serviceName := range slices.Sorted(maps.Keys(fields(services))) {
		service := fields(services)[serviceName]
		out.WriteString(commentFor(services, serviceName, "  "))
		fmt.Fprintf(&out, "  %s {\n", serviceName)
		out.WriteString(commentFor(service, "operations", "    "))
		out.WriteString("    operations {\n")
		operations := fields(fields(service)["operations"])
		for _, operationName := range slices.Sorted(maps.Keys(operations)) {
			operation := operations[operationName]
			opFields := fields(operation)
			out.WriteString(commentFor(fields(service)["operations"], operationName, "      "))
			fmt.Fprintf(&out, "      %s%s -> %s%s\n", operationName,
				convertParams(opFields["input"]), convertResult(opFields["output"]), trailingComment(operation))
		}
		out.WriteString("    }\n  }\n")
	}
	out.WriteString("}\n")
	return out.String(), nil
}

// commentFor renders the comment written above a named entry. YAML attaches it
// to the key, not to the value, so the mapping is searched by name.
func commentFor(mapping *yaml.Node, name, indent string) string {
	for _, entry := range pairs(mapping) {
		if entry.name == name {
			return commentLines(entry.key, indent)
		}
	}
	return ""
}

// serviceNode returns the mapping a service name points at, which is where
// YAML attaches the comment written above the service.
func serviceNode(services map[string]*yaml.Node, name string) *yaml.Node {
	return services[name]
}

// operationKey returns the mapping an operation name points at.
func operationKey(operations map[string]*yaml.Node, name string) *yaml.Node {
	return operations[name]
}

func convertParams(input *yaml.Node) string {
	if input == nil {
		return "()"
	}
	var parts []string
	for _, entry := range pairs(input) {
		parts = append(parts, entry.name+": "+convertFieldType(entry.node)+optionalSuffix(entry.node))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// convertResult lays out an operation result. A result whose fields are all
// scalars stays on the operation line; one that contains a record breaks onto
// its own lines. The corpus decides this, so the rule is not a preference.
func convertResult(output *yaml.Node) string {
	if output == nil {
		return "{}"
	}
	entries := pairs(output)
	if len(entries) == 0 {
		return "{}"
	}
	nested := false
	for _, entry := range entries {
		if fieldHasRecord(entry.node) {
			nested = true
		}
	}
	if !nested {
		var parts []string
		for _, entry := range entries {
			parts = append(parts, entry.name+": "+convertFieldType(entry.node))
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	}
	var out strings.Builder
	out.WriteString("{\n")
	for _, entry := range entries {
		fmt.Fprintf(&out, "        %s: %s,\n", entry.name, convertFieldType(entry.node))
	}
	out.WriteString("      }")
	return out.String()
}

// fieldHasRecord reports whether a field type contains a record, directly or as
// a list element.
func fieldHasRecord(node *yaml.Node) bool {
	meta := fields(node)
	if kind, _ := scalar(meta["type"]); kind == "list" {
		return len(fields(meta["items"])) > 0
	}
	return false
}

// convertFieldType reads the YAML field metadata: type, items, nullable, and
// optional. A default or one_of group has no beta form yet.
// convertFieldType reads one API field. It refuses anything it cannot express
// rather than dropping it: a lost one_of group or default would change the
// declaration silently.
func convertFieldType(node *yaml.Node) string {
	meta := fields(node)
	if meta["one_of"] != nil {
		return ""
	}
	kind, _ := scalar(meta["type"])
	switch kind {
	case "list":
		items := fields(meta["items"])
		if len(items) == 0 {
			return "string[]"
		}
		var parts []string
		for _, entry := range pairs(meta["items"]) {
			parts = append(parts, entry.name+": "+convertFieldType(entry.node)+optionalSuffix(entry.node))
		}
		return "{ " + strings.Join(parts, ", ") + " }[]"
	case "":
		return "string"
	}
	if meta["default"] != nil {
		return ""
	}
	result := kind
	if nullable := meta["nullable"]; nullable != nil && nullable.Value == "true" {
		result += "?"
	}
	return result
}

// optionalSuffix renders the optional marker. A field that may be omitted is
// written `name: type optional`; a nullable value is `type?`. Keeping them
// apart matters because a nullable field is still required.
func optionalSuffix(node *yaml.Node) string {
	if meta := fields(node); meta["optional"] != nil && meta["optional"].Value == "true" {
		return " optional"
	}
	return ""
}
