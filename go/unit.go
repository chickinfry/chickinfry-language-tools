package chiklang

import "strings"

// RoleForPath reports the .chik role file that owns a settings path, or an
// empty string when the path is outside the declaration language. It mirrors
// the mapping the YAML writer uses, so one path names one file in both forms.
func RoleForPath(path string) string {
	switch {
	case path == "services", hasPrefix(path, "services."), hasPrefix(path, "types."), hasPrefix(path, "routes."):
		return "api"
	case hasPrefix(path, "auth."):
		return "auth"
	case hasPrefix(path, "clients."):
		return "clients"
	case path == "project.name", path == "storage", hasPrefix(path, "storage."), path == "vault",
		path == "realtime", hasPrefix(path, "generation."):
		return "project"
	case path == "database", hasPrefix(path, "database."):
		return "schema"
	}
	return ""
}

func hasPrefix(path, prefix string) bool {
	return strings.HasPrefix(path, prefix)
}

// UnitText renders the named units of a role from a canonical YAML document.
// The caller converts the merged snapshot to YAML once, so every edited unit is
// written from the same normalized declaration and nothing is hand-formatted.
func UnitText(role string, yamlDocument []byte, paths []string) (map[string]string, []Diagnostic) {
	text, diagnostics := Convert(role, yamlDocument)
	if len(diagnostics) != 0 {
		return nil, diagnostics
	}
	file := Parse("chik/"+role+".chik", text)
	if len(file.Diagnostics) != 0 {
		return nil, file.Diagnostics
	}
	index := declarationPaths(file)
	units := make(map[string]string, len(paths))
	for _, path := range paths {
		ref, ok := index[path]
		if !ok {
			continue
		}
		units[path] = file.Source[ref.start:ref.end]
	}
	return units, nil
}
