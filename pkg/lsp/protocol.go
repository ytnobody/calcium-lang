// Package lsp implements the Language Server Protocol (LSP) for the Calcium language.
// It follows the LSP specification 3.17: https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/
package lsp

// -------------------------------------------------------------------------
// JSON-RPC 2.0 base types
// -------------------------------------------------------------------------

// RequestMessage is a JSON-RPC 2.0 request
type RequestMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  any `json:"params,omitempty"`
}

// ResponseMessage is a JSON-RPC 2.0 response
type ResponseMessage struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any   `json:"id"`
	Result  any   `json:"result,omitempty"`
	Error   *ResponseError `json:"error,omitempty"`
}

// NotificationMessage is a JSON-RPC 2.0 notification (no ID)
type NotificationMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  any `json:"params,omitempty"`
}

// ResponseError represents a JSON-RPC error object
type ResponseError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any `json:"data,omitempty"`
}

// Error codes (JSON-RPC + LSP)
const (
	ParseError           = -32700
	InvalidRequest       = -32600
	MethodNotFound       = -32601
	InvalidParams        = -32602
	InternalError        = -32603
	ServerNotInitialized = -32002
	RequestCancelled     = -32800
)

// -------------------------------------------------------------------------
// LSP basic types
// -------------------------------------------------------------------------

// Position in a text document, zero-based line and character
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range is a range in a text document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location is a location in a text document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifies a text document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentItem represents an open text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// VersionedTextDocumentIdentifier includes the version
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent describes a change to a text document
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// -------------------------------------------------------------------------
// Initialize
// -------------------------------------------------------------------------

// InitializeParams are the parameters for the initialize request
type InitializeParams struct {
	ProcessID    *int              `json:"processId"`
	RootURI      string            `json:"rootUri"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

// ClientCapabilities (minimal subset)
type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
}

// TextDocumentClientCapabilities (minimal)
type TextDocumentClientCapabilities struct {
	Completion  *CompletionClientCapabilities  `json:"completion,omitempty"`
	Hover       *HoverClientCapabilities       `json:"hover,omitempty"`
	Definition  *DefinitionClientCapabilities  `json:"definition,omitempty"`
	Diagnostics *DiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
}

// CompletionClientCapabilities (minimal)
type CompletionClientCapabilities struct {
	SnippetSupport bool `json:"snippetSupport,omitempty"`
}

// HoverClientCapabilities (minimal)
type HoverClientCapabilities struct{}

// DefinitionClientCapabilities (minimal)
type DefinitionClientCapabilities struct{}

// DiagnosticsClientCapabilities (minimal)
type DiagnosticsClientCapabilities struct{}

// InitializeResult is the result of the initialize request
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo contains information about the server
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerCapabilities describes the server's capabilities
type ServerCapabilities struct {
	TextDocumentSync   int                  `json:"textDocumentSync"`
	CompletionProvider *CompletionOptions   `json:"completionProvider,omitempty"`
	HoverProvider      bool                 `json:"hoverProvider,omitempty"`
	DefinitionProvider bool                 `json:"definitionProvider,omitempty"`
}

// CompletionOptions describes completion server capabilities
type CompletionOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// TextDocumentSyncKind
const (
	SyncNone        = 0
	SyncFull        = 1
	SyncIncremental = 2
)

// -------------------------------------------------------------------------
// Diagnostics
// -------------------------------------------------------------------------

// DiagnosticSeverity represents severity of a diagnostic
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// Diagnostic represents a diagnostic message
type Diagnostic struct {
	Range    Range              `json:"range"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Source   string             `json:"source,omitempty"`
	Message  string             `json:"message"`
}

// PublishDiagnosticsParams are the parameters for diagnostics notifications
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// -------------------------------------------------------------------------
// Completion
// -------------------------------------------------------------------------

// CompletionItemKind is the kind of completion item
type CompletionItemKind int

const (
	KindText        CompletionItemKind = 1
	KindMethod      CompletionItemKind = 2
	KindFunction    CompletionItemKind = 3
	KindConstructor CompletionItemKind = 4
	KindField       CompletionItemKind = 5
	KindVariable    CompletionItemKind = 6
	KindClass       CompletionItemKind = 7
	KindInterface   CompletionItemKind = 8
	KindModule      CompletionItemKind = 9
	KindProperty    CompletionItemKind = 10
	KindUnit        CompletionItemKind = 11
	KindValue       CompletionItemKind = 12
	KindEnum        CompletionItemKind = 13
	KindKeyword     CompletionItemKind = 14
	KindSnippet     CompletionItemKind = 15
	KindColor       CompletionItemKind = 16
	KindFile        CompletionItemKind = 17
	KindReference   CompletionItemKind = 18
	KindFolder      CompletionItemKind = 19
	KindEnumMember  CompletionItemKind = 20
	KindConstant    CompletionItemKind = 21
	KindStruct      CompletionItemKind = 22
	KindEvent       CompletionItemKind = 23
	KindOperator    CompletionItemKind = 24
	KindTypeParam   CompletionItemKind = 25
)

// CompletionItem represents a single completion suggestion
type CompletionItem struct {
	Label            string             `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string             `json:"detail,omitempty"`
	Documentation    string             `json:"documentation,omitempty"`
	InsertText       string             `json:"insertText,omitempty"`
}

// CompletionList is a list of completion items
type CompletionList struct {
	IsIncomplete bool             `json:"isIncomplete"`
	Items        []CompletionItem `json:"items"`
}

// CompletionParams are the parameters for a completion request
type CompletionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// -------------------------------------------------------------------------
// Hover
// -------------------------------------------------------------------------

// MarkupContent represents human-readable content
type MarkupContent struct {
	Kind  string `json:"kind"` // "plaintext" or "markdown"
	Value string `json:"value"`
}

// Hover is the result of a hover request
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

// HoverParams are the parameters for a hover request
type HoverParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// -------------------------------------------------------------------------
// Definition
// -------------------------------------------------------------------------

// DefinitionParams are the parameters for a definition request
type DefinitionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// -------------------------------------------------------------------------
// Text document notification params
// -------------------------------------------------------------------------

// DidOpenTextDocumentParams are the params for textDocument/didOpen
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams are the params for textDocument/didChange
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier   `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams are the params for textDocument/didClose
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
