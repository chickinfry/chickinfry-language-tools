package lsp

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// serve collects replies and the error Serve returned, so a test can look at
// both the protocol answers and the process-level result.
func serve(t *testing.T, messages ...string) ([]Message, error) {
	t.Helper()
	var input bytes.Buffer
	for _, message := range messages {
		body := []byte(message)
		input.WriteString("Content-Length: " + itoa(len(body)) + "\r\n\r\n")
		input.Write(body)
	}
	var output bytes.Buffer
	err := Serve(&input, &output)
	reader := bufio.NewReader(&output)
	var replies []Message
	for {
		message, readErr := readMessage(reader)
		if readErr != nil {
			break
		}
		replies = append(replies, *message)
	}
	return replies, err
}

// A shutdown before exit is the clean path: the process ends successfully.
func TestShutdownThenExitEndsCleanly(t *testing.T) {
	_, err := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`,
		`{"jsonrpc":"2.0","method":"exit"}`,
	)
	if err != nil {
		t.Fatalf("Serve = %v, want nil after shutdown then exit", err)
	}
}

// Exit without shutdown is the failure path LSP 3.17 names: exit code 1.
func TestExitWithoutShutdownIsAnError(t *testing.T) {
	_, err := serve(t, `{"jsonrpc":"2.0","method":"exit"}`)
	if err == nil {
		t.Fatal("Serve = nil, want ExitError when exit has no shutdown")
	}
}

// A frame whose body is not JSON must get -32700 and leave the server alive.
func TestMalformedJSONGetsAParseErrorAndTheServerSurvives(t *testing.T) {
	broken := []byte("{not json")
	var input bytes.Buffer
	input.WriteString("Content-Length: " + itoa(len(broken)) + "\r\n\r\n")
	input.Write(broken)
	// A valid request after the broken frame proves the loop continued.
	good := []byte(`{"jsonrpc":"2.0","id":7,"method":"initialize","params":{}}`)
	input.WriteString("Content-Length: " + itoa(len(good)) + "\r\n\r\n")
	input.Write(good)

	var output bytes.Buffer
	if err := Serve(&input, &output); err != nil {
		t.Fatalf("Serve = %v, want nil", err)
	}
	reader := bufio.NewReader(&output)
	first, err := readMessage(reader)
	if err != nil {
		t.Fatal(err)
	}
	if first.Error == nil || first.Error.Code != -32700 {
		t.Fatalf("first reply = %+v, want a -32700 parse error", first)
	}
	second, err := readMessage(reader)
	if err != nil {
		t.Fatalf("the server did not answer after a parse error: %v", err)
	}
	if string(second.ID) != "7" {
		t.Fatalf("second reply id = %s, want 7", second.ID)
	}
}

// A frame without Content-Length is also a parse error, not a crash.
func TestMissingContentLengthGetsAParseError(t *testing.T) {
	var input bytes.Buffer
	input.WriteString("X-Nothing: 1\r\n\r\n")
	good := []byte(`{"jsonrpc":"2.0","id":8,"method":"initialize","params":{}}`)
	input.WriteString("Content-Length: " + itoa(len(good)) + "\r\n\r\n")
	input.Write(good)

	var output bytes.Buffer
	if err := Serve(&input, &output); err != nil {
		t.Fatalf("Serve = %v, want nil", err)
	}
	reader := bufio.NewReader(&output)
	first, err := readMessage(reader)
	if err != nil {
		t.Fatal(err)
	}
	if first.Error == nil || first.Error.Code != -32700 {
		t.Fatalf("first reply = %+v, want a -32700 parse error", first)
	}
}

// After shutdown, a request is invalid; only exit is expected.
func TestRequestAfterShutdownIsInvalid(t *testing.T) {
	replies, _ := serve(t,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"shutdown"}`,
		`{"jsonrpc":"2.0","id":3,"method":"textDocument/completion","params":{}}`,
	)
	var last *Message
	for index := range replies {
		if string(replies[index].ID) == "3" {
			last = &replies[index]
		}
	}
	if last == nil {
		t.Fatal("no reply to the request after shutdown")
	}
	if last.Error == nil || last.Error.Code != -32600 {
		t.Fatalf("reply = %+v, want -32600", last)
	}
}

// A request whose params do not decode gets -32602 rather than being dropped.
func TestBadParamsGetAnInvalidParamsReply(t *testing.T) {
	replies, _ := serve(t, `{"jsonrpc":"2.0","id":4,"method":"textDocument/didOpen","params":"not an object"}`)
	if len(replies) != 1 {
		t.Fatalf("got %d replies, want 1", len(replies))
	}
	if replies[0].Error == nil || replies[0].Error.Code != -32602 {
		t.Fatalf("reply = %+v, want -32602", replies[0])
	}
}

// The message must name the code, so a reader can search for it.
func TestParseErrorMentionsTheCode(t *testing.T) {
	var input bytes.Buffer
	broken := []byte("nope")
	input.WriteString("Content-Length: " + itoa(len(broken)) + "\r\n\r\n")
	input.Write(broken)
	var output bytes.Buffer
	if err := Serve(&input, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "-32700") {
		t.Fatalf("output does not name -32700: %s", output.String())
	}
}
