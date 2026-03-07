package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
)

// writeRPCMessage encodes a JSON-RPC message with Content-Length framing
func writeRPCMessage(buf *bytes.Buffer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(buf, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return err
}

// readRPCMessage reads one framed message from a reader
func readRPCMessage(r *bufio.Reader) (map[string]json.RawMessage, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			headers[parts[0]] = parts[1]
		}
	}
	var cl int
	fmt.Sscanf(headers["Content-Length"], "%d", &cl)
	body := make([]byte, cl)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	var msg map[string]json.RawMessage
	return msg, json.Unmarshal(body, &msg)
}

// runServer is a helper that feeds input to the server and collects output
func runServer(messages []any) []map[string]json.RawMessage {
	var in bytes.Buffer
	for _, m := range messages {
		_ = writeRPCMessage(&in, m)
	}

	var out bytes.Buffer
	logger := log.New(io.Discard, "", 0)
	server := NewServer(&in, &out, logger)
	server.Run()

	// Parse all output messages
	reader := bufio.NewReader(&out)
	var results []map[string]json.RawMessage
	for {
		msg, err := readRPCMessage(reader)
		if err != nil {
			break
		}
		results = append(results, msg)
	}
	return results
}

func TestServer_Initialize(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)
	if len(responses) == 0 {
		t.Fatal("expected responses from server")
	}
	// First response should be from initialize
	resp := responses[0]
	if _, ok := resp["result"]; !ok {
		t.Error("expected 'result' in initialize response")
	}
	if _, ok := resp["error"]; ok {
		t.Error("unexpected 'error' in initialize response")
	}
}

func TestServer_DidOpen_PublishesDiagnostics(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "initialized",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "textDocument/didOpen",
			"params": map[string]any{
				"textDocument": map[string]any{
					"uri":        "file:///test.ca",
					"languageId": "calcium",
					"version":    1,
					"text":       "x = 42;",
				},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)

	// Look for publishDiagnostics notification
	found := false
	for _, resp := range responses {
		var method string
		if raw, ok := resp["method"]; ok {
			_ = json.Unmarshal(raw, &method)
		}
		if method == "textDocument/publishDiagnostics" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected textDocument/publishDiagnostics notification")
	}
}

func TestServer_Completion(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "textDocument/didOpen",
			"params": map[string]any{
				"textDocument": map[string]any{
					"uri":        "file:///test.ca",
					"languageId": "calcium",
					"version":    1,
					"text":       "func greet(name) = name;",
				},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "textDocument/completion",
			"params": map[string]any{
				"textDocument": map[string]any{"uri": "file:///test.ca"},
				"position":     map[string]any{"line": 0, "character": 3},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)

	// Find completion response (id=2)
	var completionResp map[string]json.RawMessage
	for _, resp := range responses {
		var id json.Number
		if raw, ok := resp["id"]; ok {
			_ = json.Unmarshal(raw, &id)
			if id.String() == "2" {
				completionResp = resp
				break
			}
		}
	}
	if completionResp == nil {
		t.Fatal("expected completion response")
	}
	if _, ok := completionResp["result"]; !ok {
		t.Error("expected 'result' in completion response")
	}
}

func TestServer_Hover(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "textDocument/didOpen",
			"params": map[string]any{
				"textDocument": map[string]any{
					"uri":        "file:///test.ca",
					"languageId": "calcium",
					"version":    1,
					"text":       "func add(a, b) = a + b;",
				},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "textDocument/hover",
			"params": map[string]any{
				"textDocument": map[string]any{"uri": "file:///test.ca"},
				"position":     map[string]any{"line": 0, "character": 0},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)

	// Find hover response (id=2)
	var hoverResp map[string]json.RawMessage
	for _, resp := range responses {
		var id json.Number
		if raw, ok := resp["id"]; ok {
			_ = json.Unmarshal(raw, &id)
			if id.String() == "2" {
				hoverResp = resp
				break
			}
		}
	}
	if hoverResp == nil {
		t.Fatal("expected hover response")
	}
}

func TestServer_Definition(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "textDocument/didOpen",
			"params": map[string]any{
				"textDocument": map[string]any{
					"uri":        "file:///test.ca",
					"languageId": "calcium",
					"version":    1,
					"text":       "func square(x) = x * x;\nsquare",
				},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "textDocument/definition",
			"params": map[string]any{
				"textDocument": map[string]any{"uri": "file:///test.ca"},
				"position":     map[string]any{"line": 1, "character": 0},
			},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)

	var defResp map[string]json.RawMessage
	for _, resp := range responses {
		var id json.Number
		if raw, ok := resp["id"]; ok {
			_ = json.Unmarshal(raw, &id)
			if id.String() == "2" {
				defResp = resp
				break
			}
		}
	}
	if defResp == nil {
		t.Fatal("expected definition response")
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "unknownMethod/something",
			"params":  map[string]any{},
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)

	found := false
	for _, resp := range responses {
		if errRaw, ok := resp["error"]; ok {
			var errObj map[string]json.RawMessage
			if err := json.Unmarshal(errRaw, &errObj); err == nil {
				var code int
				if raw, ok := errObj["code"]; ok {
					_ = json.Unmarshal(raw, &code)
					if code == MethodNotFound {
						found = true
					}
				}
			}
		}
	}
	if !found {
		t.Error("expected MethodNotFound error response")
	}
}

func TestServer_Shutdown(t *testing.T) {
	msgs := []any{
		map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "shutdown",
		},
		map[string]any{
			"jsonrpc": "2.0",
			"method":  "exit",
		},
	}
	responses := runServer(msgs)
	// Shutdown should return a result with no error
	found := false
	for _, resp := range responses {
		var id json.Number
		if raw, ok := resp["id"]; ok {
			_ = json.Unmarshal(raw, &id)
			if id.String() == "1" {
				found = true
				if _, ok := resp["error"]; ok {
					t.Error("unexpected error in shutdown response")
				}
			}
		}
	}
	if !found {
		t.Error("expected shutdown response")
	}
}
