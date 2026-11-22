package chiklang

import (
	"sort"
	"strings"
)

// SortDiagnostics orders diagnostics for a reader.
//
// A lexical problem comes first even when a later position sorts earlier,
// because every diagnostic after it is a consequence: an unterminated string
// swallows the rest of the line, so the parser then reports a missing token
// that is not the author's mistake. Within one stage, order is start position
// then code, which is what the error presentation spec requires.
func SortDiagnostics(diagnostics []Diagnostic) {
	rank := func(code string) int {
		switch {
		case strings.HasPrefix(code, "CHK00"):
			return 0
		default:
			return 1
		}
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		ri, rj := rank(diagnostics[i].Code), rank(diagnostics[j].Code)
		if ri != rj {
			return ri < rj
		}
		a, b := diagnostics[i].Range.Start, diagnostics[j].Range.Start
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return diagnostics[i].Code < diagnostics[j].Code
	})
}

// UniqueDiagnostics drops a repeated diagnostic, keeping the first. A parser
// that recovers from one error can reach the same token through more than one
// path, and printing the same problem several times buries the cause.
func UniqueDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	result := diagnostics[:0]
	type identity struct {
		Code, Severity string
		Range          Range
	}
	seen := make(map[identity]bool, len(diagnostics))
	for _, diagnostic := range diagnostics {
		// Wording is not part of identity: the same number at the same place is
		// the same problem even when a recovery path words it differently. The
		// path is not part of identity either: nested recovery reports one
		// missing token at the outermost and innermost enclosing path.
		key := identity{Code: diagnostic.Code, Severity: diagnostic.Severity, Range: diagnostic.Range}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, diagnostic)
	}
	return result
}
