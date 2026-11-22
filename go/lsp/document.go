package lsp

import (
	"strings"
	"unicode/utf8"
)

// document is one open file. The text is kept as the editor sent it, and the
// parse is redone on change so diagnostics always describe the current bytes.
type document struct {
	uri     string
	version int
	text    string
	lines   []string
}

func newDocument(uri string, version int, text string) *document {
	return &document{uri: uri, version: version, text: text, lines: splitLines(text)}
}

func (d *document) update(version int, text string) {
	d.version = version
	d.text = text
	d.lines = splitLines(text)
}

// splitLines keeps the trailing empty entry out, so line i is lines[i]. A
// document that ends with a newline has as many lines as it has breaks.
func splitLines(text string) []string {
	if text == "" {
		return []string{""}
	}
	trimmed := strings.TrimSuffix(text, "\n")
	return strings.Split(trimmed, "\n")
}

// position converts a one-based code point position from the parser into a
// zero-based UTF-16 position. LSP counts columns in UTF-16 code units, and a
// character outside the Basic Multilingual Plane takes two of them.
func (d *document) position(line, column int) Position {
	if line < 1 {
		line = 1
	}
	if line > len(d.lines) {
		line = len(d.lines)
	}
	lineIndex := line - 1
	runes := []rune(d.lines[lineIndex])
	index := column - 1
	if index < 0 {
		index = 0
	}
	if index > len(runes) {
		index = len(runes)
	}
	units := 0
	for _, r := range runes[:index] {
		if r >= 0x10000 {
			units += 2
			continue
		}
		units++
	}
	return Position{Line: line - 1, Character: units}
}

// offset converts a zero-based UTF-16 position into a byte offset in the text,
// which is what a range edit needs. A position past the line end clamps to the
// end, so a client that sends one cannot make the server panic.
func (d *document) offset(p Position) int {
	if p.Line < 0 {
		p.Line = 0
	}
	if p.Line >= len(d.lines) {
		return len(d.text)
	}
	offset := 0
	for index := 0; index < p.Line; index++ {
		offset += len(d.lines[index]) + 1
	}
	remaining := p.Character
	for _, r := range d.lines[p.Line] {
		width := 1
		if r >= 0x10000 {
			width = 2
		}
		if remaining < width {
			break
		}
		remaining -= width
		offset += utf8.RuneLen(r)
	}
	return offset
}

// wordAt returns the identifier around a position, or an empty string. It is
// what definition and completion key on.
func (d *document) wordAt(p Position) string {
	if p.Line < 0 || p.Line >= len(d.lines) {
		return ""
	}
	line := []rune(d.lines[p.Line])
	if len(line) == 0 {
		return ""
	}
	index := 0
	units := 0
	for index < len(line) {
		width := 1
		if line[index] >= 0x10000 {
			width = 2
		}
		if units >= p.Character {
			break
		}
		units += width
		index++
	}
	if index > len(line) {
		index = len(line)
	}
	start := index
	for start > 0 {
		previous := start - 1
		if !isWordRune(line[previous]) {
			break
		}
		start = previous
	}
	end := index
	for end < len(line) && isWordRune(line[end]) {
		end++
	}
	if start == end {
		return ""
	}
	return string(line[start:end])
}

func isWordRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
