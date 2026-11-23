// Package lsp serves .chik declarations over the Language Server Protocol.
//
// The server speaks JSON-RPC over stdio and reuses the parser, checks, and
// formatter from chiklang. It adds no language rules of its own: a diagnostic
// that appears here is the one chik check reports, and a format edit is the
// one chik fmt writes.
package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Position is a zero-based line and UTF-16 code unit offset, as LSP defines.
// The parser works in one-based code points, so every crossing converts.
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is half-open in LSP: Start is included, End is not.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a range inside a document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextEdit replaces a range with new text.
type TextEdit struct {
	Range   Range  `json:"range"`
	NewText string `json:"newText"`
}

// Diagnostic is what publishDiagnostics carries. It holds the LSP fields and
// the same content chik check --format json prints, so an agent that reads the
// JSON form finds the same code, path, help, and notes here. The extra fields
// are LSP extensions: a client ignores what it does not know, and a client that
// does know them does not need to run chik check separately.
type Diagnostic struct {
	Range      Range    `json:"range"`
	Severity   int      `json:"severity"`
	Code       string   `json:"code"`
	Source     string   `json:"source"`
	Message    string   `json:"message"`
	Path       string   `json:"path"`
	SourceLine string   `json:"source_line"`
	Help       *string  `json:"help"`
	Notes      []string `json:"notes"`
}

// CompletionItem is one suggestion.
type CompletionItem struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insertText,omitempty"`
}

// CompletionList wraps the items so the client knows the result is complete.
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// Message is one JSON-RPC envelope. Params and Result stay raw until a handler
// knows which shape to decode, which is how LSP dispatches by method.
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is the JSON-RPC error object.
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// readMessage reads one Content-Length framed message. LSP frames every
// message with HTTP-style headers, so a plain line scanner is not enough.
func readMessage(reader *bufio.Reader) (*Message, error) {
	length := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return nil, fmt.Errorf("read Content-Length: %w", err)
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, fmt.Errorf("message has no Content-Length")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	var message Message
	if err := json.Unmarshal(body, &message); err != nil {
		return nil, fmt.Errorf("decode message: %w", err)
	}
	return &message, nil
}

// writeMessage writes one framed message. The header is ASCII, so the byte
// count is the same as the character count.
func writeMessage(writer io.Writer, message any) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(writer, header); err != nil {
		return err
	}
	_, err = writer.Write(body)
	return err
}
