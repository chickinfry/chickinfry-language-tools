package chiklang

import (
	"sort"
	"strings"
)

// Serialize rebuilds the source from a parsed file.
//
// It walks the token and trivia streams in source order rather than the tree.
// The tree carries structure and ranges for tools; the streams carry every
// byte. Keeping the two apart is what makes the roundtrip exact without
// forcing every punctuation token into a node.
func Serialize(file *File) string {
	type span struct {
		start int
		text  string
	}
	spans := make([]span, 0, len(file.Tokens)+len(file.Trivia))
	for _, token := range file.Tokens {
		if token.Kind == EOF {
			continue
		}
		spans = append(spans, span{token.Start, rawToken(token)})
	}
	for _, t := range file.Trivia {
		spans = append(spans, span{t.Start, t.Text})
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var out strings.Builder
	last := 0
	for _, s := range spans {
		if s.start < last {
			continue
		}
		out.WriteString(s.text)
		last = s.start + len(s.text)
	}
	return out.String()
}

// rawToken returns the exact source spelling of a token.
func rawToken(token Token) string {
	if token.Raw != "" {
		return token.Raw
	}
	return token.Text
}
