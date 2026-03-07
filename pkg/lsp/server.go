package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// Server is the LSP server
type Server struct {
	in     *bufio.Reader
	out    io.Writer
	store  *DocumentStore
	logger *log.Logger
	done   chan struct{}
}

// NewServer creates a new LSP server reading from in and writing to out
func NewServer(in io.Reader, out io.Writer, logger *log.Logger) *Server {
	return &Server{
		in:     bufio.NewReader(in),
		out:    out,
		store:  NewDocumentStore(),
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Run processes LSP messages until the server exits
func (s *Server) Run() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		msg, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return
			}
			s.logger.Printf("read error: %v", err)
			return
		}

		s.handleMessage(msg)
	}
}

// readMessage reads one JSON-RPC message from the stream
func (s *Server) readMessage() (map[string]json.RawMessage, error) {
	// Read headers
	headers := make(map[string]string)
	for {
		line, err := s.in.ReadString('\n')
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

	// Get content length
	clStr, ok := headers["Content-Length"]
	if !ok {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	cl, err := strconv.Atoi(clStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Length: %v", err)
	}

	// Read body
	body := make([]byte, cl)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return nil, err
	}

	var msg map[string]json.RawMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// writeMessage sends a JSON-RPC message
func (s *Server) writeMessage(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		s.logger.Printf("marshal error: %v", err)
		return
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	_, _ = io.WriteString(s.out, header)
	_, _ = s.out.Write(data)
}

// sendResponse sends a JSON-RPC response
func (s *Server) sendResponse(id any, result any, respErr *ResponseError) {
	s.writeMessage(ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   respErr,
	})
}

// sendNotification sends a JSON-RPC notification (no ID)
func (s *Server) sendNotification(method string, params any) {
	s.writeMessage(NotificationMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	})
}

// handleMessage dispatches a message to the appropriate handler
func (s *Server) handleMessage(msg map[string]json.RawMessage) {
	// Extract method
	var method string
	if raw, ok := msg["method"]; ok {
		_ = json.Unmarshal(raw, &method)
	}

	// Extract ID (may be absent for notifications)
	var id any
	if raw, ok := msg["id"]; ok {
		_ = json.Unmarshal(raw, &id)
	}

	// Extract params
	var params json.RawMessage
	if raw, ok := msg["params"]; ok {
		params = raw
	}

	s.logger.Printf("-> %s", method)

	switch method {
	case "initialize":
		s.handleInitialize(id, params)
	case "initialized":
		// no-op
	case "shutdown":
		s.sendResponse(id, nil, nil)
	case "exit":
		close(s.done)
	case "textDocument/didOpen":
		s.handleDidOpen(params)
	case "textDocument/didChange":
		s.handleDidChange(params)
	case "textDocument/didClose":
		s.handleDidClose(params)
	case "textDocument/completion":
		s.handleCompletion(id, params)
	case "textDocument/hover":
		s.handleHover(id, params)
	case "textDocument/definition":
		s.handleDefinition(id, params)
	default:
		if id != nil {
			s.sendResponse(id, nil, &ResponseError{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("method not found: %s", method),
			})
		}
	}
}

// handleInitialize responds to the initialize request
func (s *Server) handleInitialize(id any, _ json.RawMessage) {
	result := InitializeResult{
		Capabilities: ServerCapabilities{
			TextDocumentSync: SyncFull,
			CompletionProvider: &CompletionOptions{
				TriggerCharacters: []string{".", "_"},
			},
			HoverProvider:      true,
			DefinitionProvider: true,
		},
		ServerInfo: &ServerInfo{
			Name:    "calcium-lsp",
			Version: "0.1.0",
		},
	}
	s.sendResponse(id, result, nil)
}

// handleDidOpen processes textDocument/didOpen
func (s *Server) handleDidOpen(params json.RawMessage) {
	var p DidOpenTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		s.logger.Printf("didOpen unmarshal error: %v", err)
		return
	}
	doc := s.store.Open(p.TextDocument.URI, p.TextDocument.Version, p.TextDocument.Text)
	s.publishDiagnostics(doc)
}

// handleDidChange processes textDocument/didChange
func (s *Server) handleDidChange(params json.RawMessage) {
	var p DidChangeTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		s.logger.Printf("didChange unmarshal error: %v", err)
		return
	}
	if len(p.ContentChanges) == 0 {
		return
	}
	// We use full sync, so take the last change
	text := p.ContentChanges[len(p.ContentChanges)-1].Text
	doc := s.store.Update(p.TextDocument.URI, p.TextDocument.Version, text)
	s.publishDiagnostics(doc)
}

// handleDidClose processes textDocument/didClose
func (s *Server) handleDidClose(params json.RawMessage) {
	var p DidCloseTextDocumentParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}
	s.store.Close(p.TextDocument.URI)
	// Clear diagnostics for closed file
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         p.TextDocument.URI,
		Diagnostics: []Diagnostic{},
	})
}

// publishDiagnostics sends parse error diagnostics for a document
func (s *Server) publishDiagnostics(doc *Document) {
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         doc.URI,
		Diagnostics: doc.Diagnostics(),
	})
}

// handleCompletion handles textDocument/completion
func (s *Server) handleCompletion(id any, params json.RawMessage) {
	var p CompletionParams
	if err := json.Unmarshal(params, &p); err != nil {
		s.sendResponse(id, CompletionList{Items: []CompletionItem{}}, nil)
		return
	}
	doc, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		s.sendResponse(id, CompletionList{Items: []CompletionItem{}}, nil)
		return
	}
	items := doc.CompletionAt(p.Position)
	s.sendResponse(id, CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil)
}

// handleHover handles textDocument/hover
func (s *Server) handleHover(id any, params json.RawMessage) {
	var p HoverParams
	if err := json.Unmarshal(params, &p); err != nil {
		s.sendResponse(id, nil, nil)
		return
	}
	doc, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		s.sendResponse(id, nil, nil)
		return
	}
	hover := doc.HoverAt(p.Position)
	s.sendResponse(id, hover, nil)
}

// handleDefinition handles textDocument/definition
func (s *Server) handleDefinition(id any, params json.RawMessage) {
	var p DefinitionParams
	if err := json.Unmarshal(params, &p); err != nil {
		s.sendResponse(id, nil, nil)
		return
	}
	doc, ok := s.store.Get(p.TextDocument.URI)
	if !ok {
		s.sendResponse(id, nil, nil)
		return
	}
	loc := doc.DefinitionAt(p.Position)
	s.sendResponse(id, loc, nil)
}
