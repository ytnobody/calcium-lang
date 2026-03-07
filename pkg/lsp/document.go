package lsp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ytnobody/calcium-lang/pkg/ast"
	"github.com/ytnobody/calcium-lang/pkg/lexer"
	"github.com/ytnobody/calcium-lang/pkg/parser"
	"github.com/ytnobody/calcium-lang/pkg/token"
)

// Document holds the state for an open document
type Document struct {
	URI     string
	Version int
	Text    string
	// Parsed state (may be nil if not yet parsed)
	Program     *ast.Program
	ParseErrors []string
	// Symbol table built from the AST
	Symbols map[string]*SymbolInfo
}

// SymbolInfo stores information about a defined symbol
type SymbolInfo struct {
	Name   string
	Kind   SymbolKind
	Line   int // 1-based
	Column int // 1-based
	Detail string
}

// SymbolKind distinguishes what kind of symbol it is
type SymbolKind int

const (
	SymbolVariable SymbolKind = iota
	SymbolFunction
	SymbolConstraint
	SymbolType
	SymbolNamespace
)

// DocumentStore manages open documents
type DocumentStore struct {
	docs map[string]*Document
}

// NewDocumentStore creates an empty document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]*Document)}
}

// Open stores a new document and parses it
func (ds *DocumentStore) Open(uri string, version int, text string) *Document {
	doc := &Document{URI: uri, Version: version, Text: text}
	doc.parse()
	ds.docs[uri] = doc
	return doc
}

// Update replaces a document's text and re-parses
func (ds *DocumentStore) Update(uri string, version int, text string) *Document {
	doc, ok := ds.docs[uri]
	if !ok {
		return ds.Open(uri, version, text)
	}
	doc.Version = version
	doc.Text = text
	doc.parse()
	return doc
}

// Close removes a document from the store
func (ds *DocumentStore) Close(uri string) {
	delete(ds.docs, uri)
}

// Get retrieves a document by URI
func (ds *DocumentStore) Get(uri string) (*Document, bool) {
	doc, ok := ds.docs[uri]
	return doc, ok
}

// parse lexes + parses the document and builds the symbol table
func (d *Document) parse() {
	l := lexer.New(d.Text)
	p := parser.New(l)
	d.Program = p.ParseProgram()
	d.ParseErrors = p.Errors()
	d.buildSymbols()
}

// buildSymbols walks the AST and collects defined symbols.
// It uses recover() because the parser can return partially-constructed
// AST nodes (e.g. nil-pointer structs wrapped in interfaces) when
// the source has syntax errors.
func (d *Document) buildSymbols() {
	d.Symbols = make(map[string]*SymbolInfo)
	if d.Program == nil {
		return
	}
	defer func() {
		// Recover from any nil-pointer panics that may occur when
		// walking partially-parsed AST nodes from erroneous source.
		recover() //nolint:errcheck
	}()
	for _, stmt := range d.Program.Statements {
		if stmt == nil {
			continue
		}
		switch s := stmt.(type) {
		case *ast.AssignmentStatement:
			if s.Name == nil {
				continue
			}
			d.Symbols[s.Name.Value] = &SymbolInfo{
				Name:   s.Name.Value,
				Kind:   SymbolVariable,
				Line:   s.Token.Line,
				Column: s.Token.Column,
				Detail: "variable",
			}
		case *ast.FunctionDeclaration:
			if s.Name == nil {
				continue
			}
			kind := SymbolFunction
			params := make([]string, 0, len(s.Parameters))
			for _, p := range s.Parameters {
				if p != nil {
					params = append(params, p.Value)
				}
			}
			sig := fmt.Sprintf("func %s(%s)", s.Name.Value, strings.Join(params, ", "))
			if s.IsEffect {
				sig = "func! " + s.Name.Value + "(" + strings.Join(params, ", ") + ")"
			}
			d.Symbols[s.Name.Value] = &SymbolInfo{
				Name:   s.Name.Value,
				Kind:   kind,
				Line:   s.Token.Line,
				Column: s.Token.Column,
				Detail: sig,
			}
		case *ast.ConstraintDeclaration:
			if s.Name == nil {
				continue
			}
			params := make([]string, 0, len(s.Parameters))
			for _, p := range s.Parameters {
				if p != nil {
					params = append(params, p.Value)
				}
			}
			d.Symbols[s.Name.Value] = &SymbolInfo{
				Name:   s.Name.Value,
				Kind:   SymbolConstraint,
				Line:   s.Token.Line,
				Column: s.Token.Column,
				Detail: fmt.Sprintf("constraint %s(%s)", s.Name.Value, strings.Join(params, ", ")),
			}
		case *ast.TypeDeclaration:
			if s.Name == nil {
				continue
			}
			d.Symbols[s.Name.Value] = &SymbolInfo{
				Name:   s.Name.Value,
				Kind:   SymbolType,
				Line:   s.Token.Line,
				Column: s.Token.Column,
				Detail: fmt.Sprintf("type %s", s.Name.Value),
			}
			// Also add variant constructors
			for _, v := range s.Variants {
				if v == nil {
					continue
				}
				d.Symbols[v.Name] = &SymbolInfo{
					Name:   v.Name,
					Kind:   SymbolFunction,
					Line:   s.Token.Line,
					Column: s.Token.Column,
					Detail: fmt.Sprintf("%s constructor of type %s", v.Name, s.Name.Value),
				}
			}
		case *ast.NamespaceDeclaration:
			if s.Name == nil {
				continue
			}
			name := s.Name.String()
			d.Symbols[name] = &SymbolInfo{
				Name:   name,
				Kind:   SymbolNamespace,
				Line:   s.Token.Line,
				Column: s.Token.Column,
				Detail: fmt.Sprintf("namespace %s", name),
			}
		}
	}
}

// Diagnostics converts parse errors into LSP Diagnostic items.
// Parse error strings contain line/column info embedded in the message.
// We try to extract them; otherwise we default to (0,0).
func (d *Document) Diagnostics() []Diagnostic {
	if len(d.ParseErrors) == 0 {
		return []Diagnostic{}
	}
	diags := make([]Diagnostic, 0, len(d.ParseErrors))
	for _, e := range d.ParseErrors {
		line, col := extractPosition(e)
		// LSP uses 0-based line/character
		if line > 0 {
			line--
		}
		if col > 0 {
			col--
		}
		pos := Position{Line: line, Character: col}
		diag := Diagnostic{
			Range: Range{
				Start: pos,
				End:   Position{Line: pos.Line, Character: pos.Character + 1},
			},
			Severity: SeverityError,
			Source:   "calcium",
			Message:  e,
		}
		diags = append(diags, diag)
	}
	return diags
}

// extractPosition tries to extract line/col from an error message.
// Error messages typically contain "line X, column Y" or "[line:col]".
var (
	reLine    = regexp.MustCompile(`[Ll]ine\s+(\d+)`)
	reLineCol = regexp.MustCompile(`\[(\d+):(\d+)\]`)
	reAtLine  = regexp.MustCompile(`at line (\d+)(?:, col(?:umn)? (\d+))?`)
)

func extractPosition(msg string) (line, col int) {
	if m := reLineCol.FindStringSubmatch(msg); len(m) == 3 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			line = v
		}
		if v, err := strconv.Atoi(m[2]); err == nil {
			col = v
		}
		return
	}
	if m := reAtLine.FindStringSubmatch(msg); len(m) >= 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			line = v
		}
		if len(m) >= 3 && m[2] != "" {
			if v, err := strconv.Atoi(m[2]); err == nil {
				col = v
			}
		}
		return
	}
	if m := reLine.FindStringSubmatch(msg); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil {
			line = v
		}
		return
	}
	return 0, 0
}

// WordAtPosition returns the word (identifier) at the given 0-based position in the document.
func (d *Document) WordAtPosition(pos Position) string {
	lines := strings.Split(d.Text, "\n")
	if pos.Line >= len(lines) {
		return ""
	}
	line := lines[pos.Line]
	ch := min(pos.Character, len(line))
	// Find start of word
	start := ch
	for start > 0 && isIdentChar(rune(line[start-1])) {
		start--
	}
	// Find end of word
	end := ch
	for end < len(line) && isIdentChar(rune(line[end])) {
		end++
	}
	return line[start:end]
}

func isIdentChar(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '!'
}

// CompletionAt returns completion items at the given position
func (d *Document) CompletionAt(pos Position) []CompletionItem {
	word := d.WordAtPosition(pos)
	items := make([]CompletionItem, 0)

	// Add keyword completions
	for _, kw := range token.GetAllKeywords() {
		if word == "" || strings.HasPrefix(kw, word) {
			items = append(items, CompletionItem{
				Label:  kw,
				Kind:   KindKeyword,
				Detail: "keyword",
			})
		}
	}

	// Add built-in functions
	for _, bi := range builtinCompletions {
		if word == "" || strings.HasPrefix(bi.Label, word) {
			items = append(items, bi)
		}
	}

	// Add symbols from the document
	for name, sym := range d.Symbols {
		if word == "" || strings.HasPrefix(name, word) {
			kind := KindVariable
			switch sym.Kind {
			case SymbolFunction:
				kind = KindFunction
			case SymbolConstraint:
				kind = KindInterface
			case SymbolType:
				kind = KindClass
			case SymbolNamespace:
				kind = KindModule
			}
			items = append(items, CompletionItem{
				Label:  name,
				Kind:   CompletionItemKind(kind),
				Detail: sym.Detail,
			})
		}
	}

	return items
}

// HoverAt returns hover information at the given position
func (d *Document) HoverAt(pos Position) *Hover {
	word := d.WordAtPosition(pos)
	if word == "" {
		return nil
	}

	// Check built-in docs first
	if doc, ok := builtinDocs[word]; ok {
		return &Hover{
			Contents: MarkupContent{Kind: "markdown", Value: doc},
		}
	}

	// Check document symbols
	if sym, ok := d.Symbols[word]; ok {
		return &Hover{
			Contents: MarkupContent{Kind: "markdown", Value: fmt.Sprintf("```calcium\n%s\n```", sym.Detail)},
		}
	}

	return nil
}

// DefinitionAt returns the location of the definition of the symbol at pos
func (d *Document) DefinitionAt(pos Position) *Location {
	word := d.WordAtPosition(pos)
	if word == "" {
		return nil
	}
	sym, ok := d.Symbols[word]
	if !ok {
		return nil
	}
	// Convert 1-based to 0-based
	line := sym.Line - 1
	col := sym.Column - 1
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	return &Location{
		URI: d.URI,
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col + len(word)},
		},
	}
}

// -------------------------------------------------------------------------
// Built-in completions and docs
// -------------------------------------------------------------------------

var builtinCompletions = []CompletionItem{
	{Label: "len", Kind: KindFunction, Detail: "len(collection) -> Int", Documentation: "Returns the length of a collection"},
	{Label: "map", Kind: KindFunction, Detail: "map(collection, fn) -> Array", Documentation: "Applies fn to each element"},
	{Label: "filter", Kind: KindFunction, Detail: "filter(collection, fn) -> Array", Documentation: "Keeps elements where fn returns true"},
	{Label: "reduce", Kind: KindFunction, Detail: "reduce(collection, initial, fn) -> Any", Documentation: "Reduces a collection to a single value"},
	{Label: "has", Kind: KindFunction, Detail: "has(hash, key) -> Bool", Documentation: "Checks if hash contains key"},
	{Label: "keys", Kind: KindFunction, Detail: "keys(hash) -> Array", Documentation: "Returns all keys of a hash"},
	{Label: "values", Kind: KindFunction, Detail: "values(hash) -> Array", Documentation: "Returns all values of a hash"},
	{Label: "hash", Kind: KindFunction, Detail: "hash(array) -> Hash", Documentation: "Converts an array of pairs to a hash"},
	{Label: "success", Kind: KindFunction, Detail: "success(value) -> Result", Documentation: "Creates a successful result"},
	{Label: "failure", Kind: KindFunction, Detail: "failure(error) -> Result", Documentation: "Creates a failed result"},
	{Label: "true", Kind: KindValue, Detail: "Bool", Documentation: "Boolean true"},
	{Label: "false", Kind: KindValue, Detail: "Bool", Documentation: "Boolean false"},
}

var builtinDocs = map[string]string{
	"func":       "```calcium\nfunc name(params) = body\n```\nDeclares a pure function.",
	"func!":      "```calcium\nfunc! name(params) = body\n```\nDeclares an effect (side-effecting) function.",
	"match":      "```calcium\nmatch expr\n  pattern => result\n  _ => default\n```\nPattern matching expression.",
	"constraint": "```calcium\nconstraint Name(x) = expr\n```\nDefines a constraint (type predicate).",
	"namespace":  "```calcium\nnamespace module.path\n```\nDeclares the namespace for this file.",
	"use":        "```calcium\nuse module.path\n```\nImports a module.",
	"type":       "```calcium\ntype Name = Variant1(fields) | Variant2\n```\nDeclares an algebraic data type.",
	"do":         "```calcium\ndo\n  stmt1;\n  stmt2;\n  finalExpr\nend\n```\nBlock expression with sequential statements.",
	"return":     "```calcium\nreturn expr\n```\nExplicit return from a function body.",
	"success":    "```calcium\nsuccess(value)\n```\nCreates a successful result value.",
	"failure":    "```calcium\nfailure(error)\n```\nCreates a failed result value.",
	"in":         "Used in `for`-style expressions and range iteration.",
	"map":        "```calcium\nmap(collection, fn)\n```\nApplies `fn` to each element of the collection.",
	"filter":     "```calcium\nfilter(collection, fn)\n```\nReturns elements for which `fn` returns true.",
	"reduce":     "```calcium\nreduce(collection, initial, fn)\n```\nFolds the collection into a single value.",
	"has":        "```calcium\nhas(hash, key)\n```\nReturns true if `hash` contains `key`.",
	"keys":       "```calcium\nkeys(hash)\n```\nReturns all keys of the hash.",
	"values":     "```calcium\nvalues(hash)\n```\nReturns all values of the hash.",
	"hash":       "```calcium\nhash(array)\n```\nConverts an array of key-value pairs into a hash.",
	"len":        "```calcium\nlen(collection)\n```\nReturns the number of elements.",
	"true":       "Boolean literal `true`.",
	"false":      "Boolean literal `false`.",
}
