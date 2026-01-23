# Regex Compile-time Optimization Specification

## Overview

Pre-compile regex literals during bytecode compilation to eliminate runtime overhead.

## Design Principles

1. **Compile-time processing**: Regex is pre-compiled with Go's `regexp.Compile()` during Calcium compilation
2. **Constant pool storage**: Compiled `*regexp.Regexp` stored as constant
3. **Early syntax error detection**: Invalid regex causes compile-time error
4. **Zero runtime cost**: Pre-compiled object is immediately usable at runtime

---

## Syntax

### Regex Literal

```calcium
// Basic form
pattern = /^[a-z]+$/;

// With flags
pattern = /hello/i;        // Case insensitive
pattern = /^line$/m;       // Multiline

// Escaping
pattern = /https?:\/\//;   // Escape slashes
pattern = /\d{3}-\d{4}/;   // Digit pattern
```

### Flags

| Flag | Meaning | Go Equivalent |
|------|---------|---------------|
| `i` | Case insensitive | `(?i)` prefix |
| `m` | Multiline | `(?m)` prefix |
| `s` | `.` matches newline | `(?s)` prefix |

---

## Lexer Changes

### New Token

```go
// token/token.go
const (
    REGEX  = "REGEX"   // /pattern/flags
)

type Token struct {
    Type    TokenType
    Literal string
    // Additional info for regex
    RegexPattern string  // Pattern part
    RegexFlags   string  // Flags part
    Line    int
    Column  int
}
```

### Lexer Logic

```go
// lexer/lexer.go
func (l *Lexer) NextToken() token.Token {
    // ...
    case '/':
        if l.isRegexContext() {
            return l.readRegex()
        }
        // Division operator
        return newToken(token.SLASH, l.ch, ...)
}

func (l *Lexer) isRegexContext() bool {
    // Look at previous token to determine regex vs division
    // Regex context:
    //   - Start of statement
    //   - After =
    //   - After (
    //   - After ,
    //   - After |>
    //   - After operators
    // Division context:
    //   - After identifier
    //   - After number
    //   - After )
    //   - After ]
}

func (l *Lexer) readRegex() token.Token {
    // Read /pattern/flags
    // 1. Consume opening /
    // 2. Read until closing / (watch for \/ escape)
    // 3. Read flag characters [imsg]*
    pattern := l.readUntilUnescaped('/')
    flags := l.readRegexFlags()
    return token.Token{
        Type:         token.REGEX,
        Literal:      "/" + pattern + "/" + flags,
        RegexPattern: pattern,
        RegexFlags:   flags,
    }
}
```

---

## Parser Changes

### AST Node

```go
// ast/ast.go
type RegexLiteral struct {
    Token   token.Token
    Pattern string
    Flags   string
}

func (rl *RegexLiteral) expressionNode()      {}
func (rl *RegexLiteral) TokenLiteral() string { return rl.Token.Literal }
func (rl *RegexLiteral) String() string       { return "/" + rl.Pattern + "/" + rl.Flags }
```

### Parse Processing

```go
// parser/parser.go
func (p *Parser) parseRegexLiteral() ast.Expression {
    return &ast.RegexLiteral{
        Token:   p.curToken,
        Pattern: p.curToken.RegexPattern,
        Flags:   p.curToken.RegexFlags,
    }
}

func init() {
    // Register in prefix parse functions
    p.registerPrefix(token.REGEX, p.parseRegexLiteral)
}
```

---

## Value Type

### New Type

```go
// value/value.go
const (
    TYPE_REGEX = "regex"
)

type Value struct {
    Type ValueType
    // ... existing fields ...
    regex *regexp.Regexp  // Compiled regex
}

func Regex(re *regexp.Regexp) Value {
    return Value{Type: TYPE_REGEX, regex: re}
}

func (v Value) AsRegex() *regexp.Regexp {
    return v.regex
}

func (v Value) String() string {
    if v.Type == TYPE_REGEX {
        return v.regex.String()
    }
    // ...
}
```

---

## Compiler Changes

### Compile-time Processing

```go
// compiler/compiler.go
func (c *Compiler) Compile(node ast.Node) error {
    switch node := node.(type) {

    case *ast.RegexLiteral:
        // Convert flags to Go format
        goPattern := convertFlags(node.Pattern, node.Flags)

        // Compile regex at compile time
        re, err := regexp.Compile(goPattern)
        if err != nil {
            return fmt.Errorf("invalid regex /%s/: %s", node.Pattern, err)
        }

        // Add compiled regex to constant pool
        idx := c.addConstant(value.Regex(re))
        c.emit(bytecode.OpConstant, idx)
    }
}

func convertFlags(pattern, flags string) string {
    // Convert Calcium flags to Go (?flags) prefix
    prefix := ""
    for _, f := range flags {
        switch f {
        case 'i':
            prefix += "i"
        case 'm':
            prefix += "m"
        case 's':
            prefix += "s"
        }
    }
    if prefix != "" {
        return "(?" + prefix + ")" + pattern
    }
    return pattern
}
```

### Error Example

```
Error: invalid regex /[unclosed/: error parsing regexp: missing closing ]
  at line 5, column 10
```

---

## VM Changes

### Constant Load

Just load `TYPE_REGEX` from constant pool (no special handling needed).

```go
case bytecode.OpConstant:
    constIndex := // ...
    c := vm.constants[constIndex]
    vm.push(c)  // regex is pushed as-is
```

---

## Built-in Functions

### matches(string, regex) → bool

```go
// vm/vm.go
func builtinMatches(args ...value.Value) value.Value {
    if len(args) != 2 {
        return value.Error("matches requires 2 arguments")
    }

    str := args[0]
    re := args[1]

    if str.Type != value.TYPE_STRING {
        return value.Error("first argument must be string")
    }
    if re.Type != value.TYPE_REGEX {
        return value.Error("second argument must be regex")
    }

    // Pre-compiled, so immediate matching
    matched := re.AsRegex().MatchString(str.AsString())
    return value.Bool(matched)
}
```

### Usage Example

```calcium
email = "test@example.com";
email |> matches(/^.+@.+\..+$/);  // true

// In pipeline
input
    |> matches(/^\d{3}-\d{4}$/)
    |> validate;
```

---

## Additional Built-in Functions (Future)

| Function | Description |
|----------|-------------|
| `matches(s, re)` | Check if matches → bool |
| `find(s, re)` | Return first match → success/failure |
| `find_all(s, re)` | Return all matches as array |
| `replace(s, re, replacement)` | Replace matches |
| `split(s, re)` | Split by regex |
| `capture(s, re)` | Return capture groups as array |

---

## Implementation Steps

### Phase 1: Basic Implementation

1. [x] Add `token.REGEX` token
2. [x] Implement regex literal reading in lexer
3. [x] Add `ast.RegexLiteral` node
4. [x] Implement regex parsing in parser
5. [x] Add `value.TYPE_REGEX`
6. [x] Implement compile-time compilation in compiler
7. [x] Implement `matches` built-in function
8. [x] Tests

### Phase 2: Flag Support

1. [x] Implement `i`, `m`, `s` flags
2. [x] Flag → Go prefix conversion

### Phase 3: Additional Functions

1. [x] `find`, `find_all`
2. [x] `replace`
3. [x] `split` (regex version)
4. [x] `capture`

---

## Test Cases

```calcium
// Basic match
"hello" |> matches(/hello/);           // true
"HELLO" |> matches(/hello/);           // false
"HELLO" |> matches(/hello/i);          // true

// Patterns
"123-4567" |> matches(/^\d{3}-\d{4}$/); // true
"12-4567" |> matches(/^\d{3}-\d{4}$/);  // false

// Escaping
"https://example.com" |> matches(/https?:\/\//);  // true

// Compile error
pattern = /[invalid/;  // Compile-time error: missing closing ]
```

---

## Performance Comparison

```
Runtime compilation method:
  1000 iterations: ~50ms (regexp.Compile each time)

Compile-time method:
  1000 iterations: ~2ms (using pre-compiled)

Approximately 25x faster
```

---

## Notes

### Distinguishing from Division

```calcium
a = 10 / 2;        // Division
b = /pattern/;     // Regex
c = x / y / z;     // Division / division
d = f(/pat/, /tern/);  // Two regex as function arguments
```

Lexer looks at previous token to determine:
- After value (identifier, number, `)`, `]`) → division
- Otherwise → regex

### Dynamic Patterns

```calcium
// Not supported (not determined at compile time)
pattern = "user-" + id;
matches(input, pattern);  // Error: regex expected, got string

// Use function instead
matches(input, /user-\d+/);  // OK
```

For dynamic patterns, consider adding `regex(string)` function in the future.
