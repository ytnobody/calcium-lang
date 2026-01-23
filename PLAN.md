# Calcium Compiler Implementation Plan (Go)

## Overview

Implement the Calcium language compiler in Go. Adopt a bytecode + VM architecture, implementing the MVP (core features) first.

## Architecture

```
Source Code (.ca)
    ↓
  Lexer (Lexical Analysis)
    ↓
  Token Stream
    ↓
  Parser (Syntax Analysis)
    ↓
  AST (Abstract Syntax Tree)
    ↓
  Analyzer (Semantic Analysis)
    ↓
  Compiler (Code Generation)
    ↓
  Bytecode
    ↓
  VM (Virtual Machine)
    ↓
  Execution Result
```

## Directory Structure

```
calcium/
├── cmd/
│   └── calcium/
│       └── main.go          # CLI entry point
├── pkg/
│   ├── token/
│   │   └── token.go         # Token definitions
│   ├── lexer/
│   │   └── lexer.go         # Lexical analyzer
│   ├── ast/
│   │   └── ast.go           # AST node definitions
│   ├── parser/
│   │   └── parser.go        # Parser
│   ├── analyzer/
│   │   └── analyzer.go      # Semantic analysis (constraint checks, etc.)
│   ├── compiler/
│   │   └── compiler.go      # Bytecode generation
│   ├── bytecode/
│   │   └── bytecode.go      # Bytecode definitions
│   ├── vm/
│   │   └── vm.go            # Virtual machine
│   ├── value/
│   │   └── value.go         # Runtime value definitions
│   └── builtin/
│       └── builtin.go       # Built-in functions
├── examples/
│   └── hello.ca             # Sample programs
├── go.mod
└── go.sum
```

## Features to Implement in MVP

### Phase 1: Basic Structure ✅ Complete
- [x] Token definitions
- [x] Lexer (lexical analysis)
- [x] Basic AST structure
- [x] Parser (syntax analysis)

### Phase 2: Values and Expressions ✅ Complete
- [x] Value types (booleans, numbers, strings, arrays, functions, success/failure)
- [x] Variable bindings
- [x] Arithmetic operators (+, -, *, /, %, **)
- [x] Comparison operators (==, !=, <, >, <=, >=)
- [x] Logical operators (&&, ||, !)
- [x] Array literals and index access
- [x] Array literal concatenation syntax (`[a b]` = concatenation, `[a, b]` = individual elements)

### Phase 3: Functions ✅ Complete
- [x] Pure function definition (func)
- [x] Effect function definition (func!)
- [x] Lambda expressions ((x) => expr)
- [x] Function calls
- [x] Spread operator (`...`)
- [x] Pure pipeline operator (|>)
- [x] Effect pipeline operator (!>)
- [x] Error handling (!?)
- [x] Recursive calls

### Phase 4: Control Structures ✅ Complete
- [x] match expression
- [x] Wildcard (_) → substituted with `true`
- [x] Condition expression patterns (n > 0 => ...)

### Phase 5: Constraints ✅ Complete
- [x] Constraint definition (constraint) → implemented as functions
- [x] Constraint checking (|> Constraint?)
- [x] Parameter constraints (param: Constraint?)

### Phase 6: Built-in Functions ✅ Complete
- [x] len
- [x] concat, to_string
- [x] get, has, head, tail, push
- [x] map, filter, reduce
- [x] range

### Phase 7: Module System ✅ Complete
- [x] namespace definition (parsed)
- [x] use statement imports
- [x] Module member access (module.func)

### Phase 8: Standard Library ✅ Complete
- [x] core.io: print, println, format, read_file, write_file
- [x] core.math: abs, min, max, floor, ceil, round, sqrt, pow
- [x] core.string: split, join, trim, upper, lower, starts_with, ends_with, contains, replace, substring, char_at, index_of
- [x] core.array: reverse, slice, index_of, flatten, unique, zip, take, drop, sum, product

---

## MVP Complete! (All 42 tests PASS)

---

## Post-MVP Extensions

### Extension 1: Hashes (Associative Arrays) ✅ Complete
- [x] Hash literal `{key: value, ...}`
- [x] Dot access `hash.key`
- [x] Bracket access `hash["key"]`
- [x] keys, values functions

### Extension 2: Array Destructuring ✅ Complete
- [x] `[head | tail] = arr`
- [x] `[a, b, c] = arr`

### Extension 3: Asynchronous Processing
- [ ] core.async! (async.stay, async.spawn, async.expects, etc.)
- [ ] core.schedule! (schedule.timeout, schedule.interval)
- [ ] io.stdin, io.eof events

---

## Detailed Design

### Bytecode Instruction Set (MVP)

```
// Stack operations
CONST idx        // Push constant
POP              // Discard stack top
DUP              // Duplicate stack top

// Variables
LOAD_LOCAL idx   // Load local variable
STORE_LOCAL idx  // Store to local variable
LOAD_GLOBAL name // Load global variable
STORE_GLOBAL name// Store to global variable

// Arithmetic
ADD, SUB, MUL, DIV, MOD, POW
NEG              // Unary minus

// Comparison
EQ, NEQ, LT, GT, LTE, GTE

// Logic
AND, OR, NOT

// Control
JUMP offset      // Unconditional jump
JUMP_IF_FALSE offset  // Jump if false
RETURN           // Return from function

// Functions
CALL argc        // Function call
CALL_SPREAD argc // Function call (with spread arguments)
CLOSURE idx      // Create closure (※Since Calcium prohibits closures, simple function object)
SPREAD           // Expand array to argument list (postfix ... operator)

// Arrays
ARRAY len        // Create array (comma-separated)
ARRAY_CONCAT len // Create array + flatten (space-separated)
INDEX            // Array access

// Constraints
CHECK_CONSTRAINT // Constraint check

// Built-ins
BUILTIN name argc // Built-in function call

// Effects / Error handling
CALL_EFFECT argc  // Effect function call (func!)
WRAP_SUCCESS      // Wrap value in success()
WRAP_FAILURE      // Wrap value in failure()
UNWRAP_OR_JUMP offset // If success unwrap, if failure jump
MATCH_RESULT      // success/failure pattern match
```

### Internal Value Representation

```go
type ValueType int

const (
    VAL_BOOL ValueType = iota
    VAL_NUMBER
    VAL_STRING
    VAL_ARRAY
    VAL_FUNCTION
    VAL_SUCCESS
    VAL_FAILURE
    VAL_HASH
)

type Value struct {
    Type ValueType
    Data interface{}
}

type Function struct {
    Name       string
    Params     []string
    Bytecode   []byte
    Constants  []Value
    IsEffect   bool       // true = func! (effect function)
}
```

---

## Implementation Steps

### Step 1: Project Initialization
```bash
mkdir -p calcium/cmd/calcium calcium/pkg/{token,lexer,ast,parser,analyzer,compiler,bytecode,vm,value,builtin} examples
cd calcium
go mod init github.com/example/calcium
```

### Step 2: Tokens and Lexer
1. `pkg/token/token.go` - Define all token types
   - Keywords: `func`, `func!`, `match`, `constraint`, `namespace`, `use`, `true`, `false`, `success`, `failure`, `return`, `in`, `map`, `filter`, `reduce`, `has`, `keys`, `values`, `hash`, `len`
   - Operators: `+`, `-`, `*`, `/`, `%`, `**`, `|>`, `!>`, `!?`, `=>`, `&&`, `||`, `!`, `...`
   - Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`
   - Delimiters: `(`, `)`, `[`, `]`, `{`, `}`, `,`, `;`, `:`, `|`
   - Literals: numbers, strings, identifiers

2. `pkg/lexer/lexer.go` - Convert source code to token stream
   - Skip whitespace and comments
   - Parse numeric literals (integers, floats, hexadecimal, binary, scientific notation)
   - Parse string literals (with escape sequence support)
   - Distinguish identifiers from keywords
   - Handle compound operators (`|>`, `!>`, `!?`, `=>`, `**`, `&&`, `||`, `==`, `!=`, `<=`, `>=`, `...`)

### Step 3: AST and Parser
1. `pkg/ast/ast.go` - Expression and statement node definitions
   - Expression nodes: NumberLiteral, StringLiteral, BoolLiteral, Identifier, BinaryExpr, UnaryExpr, CallExpr, IndexExpr, ArrayLiteral, LambdaExpr, MatchExpr, PipeExpr, EffectPipeExpr, SpreadExpr
   - Statement nodes: ExpressionStmt, AssignmentStmt, FuncDecl, EffectFuncDecl, ConstraintDecl
   - ArrayLiteral records delimiter between elements (comma vs space)

2. `pkg/parser/parser.go` - Pratt parser for operator precedence
   - Define precedence table
   - Implement prefix/infix parse functions
   - Array literal parsing:
     - `[a, b]` → parsed as individual elements
     - `[a b]` → parsed as concatenation (flattening)
   - Error recovery

### Step 4: Bytecode and Compiler
1. `pkg/bytecode/bytecode.go` - Instruction definitions
   - Define OpCode constants
   - Instruction encode/decode functions

2. `pkg/compiler/compiler.go` - Generate bytecode from AST
   - Scope management
   - Constant table
   - Jump address resolution

### Step 5: VM and Execution
1. `pkg/value/value.go` - Runtime values
   - Value struct
   - Type conversion helpers

2. `pkg/vm/vm.go` - Execute bytecode
   - Stack machine implementation
   - Call frame management
   - Built-in function dispatch

3. `pkg/builtin/builtin.go` - Built-in functions
   - len, concat, to_string
   - map, filter, reduce
   - get, has

4. `pkg/stdlib/io/io.go` - core.io! module
   - io.say, io.print

### Step 6: CLI
1. `cmd/calcium/main.go` - Load and run files
   - Command line argument parsing
   - File loading
   - Error handling and display

---

## Test Strategy

Create unit tests for each phase:

```
pkg/lexer/lexer_test.go      # Tokenization tests
pkg/parser/parser_test.go    # AST generation tests
pkg/compiler/compiler_test.go # Bytecode generation tests
pkg/vm/vm_test.go            # Execution result tests
```

### Test Case Examples

**Lexer Tests:**
- Numeric literals (integers, floats, negative numbers)
- String literals (including escape sequences)
- Operators (single character, compound)
- Distinguishing keywords from identifiers

**Parser Tests:**
- Operator precedence
- Parentheses changing precedence
- Function definitions and calls
- match expressions

**VM Tests:**
- Arithmetic operations
- Variable binding and reference
- Function calls and recursion
- Pipeline operations

---

## Verification Methods

```bash
# Build
go build -o calcium ./cmd/calcium

# Run sample
./calcium examples/hello.ca

# Test
go test ./...

# Coverage
go test -cover ./...
```

---

## Sample Program (examples/hello.ca)

```calcium
use core.io!

// Hello World
"Hello, Calcium!" !> io.say;

// Variables and operations
x = 10;
y = 20;
concat("x + y = ", to_string(x + y)) !> io.say;

// Pure function definition
func double(n) = n * 2;
double(21) !> io.say;

// Pipeline
result = [1, 2, 3, 4, 5]
  |> map(x => x * 2)
  |> filter(x => x > 5)
  |> reduce((a, b) => a + b, 0);
result !> io.say;

// match
func fizzbuzz(n) =
  match n
    n % 15 == 0 => "FizzBuzz"
    n % 3 == 0 => "Fizz"
    n % 5 == 0 => "Buzz"
    _ => to_string(n);

fizzbuzz(15) !> io.say;

// Effect functions and !> pipeline
func! save_data(data) = /* Save data */;
func! notify_user(result) = /* Notify user */;

func! process_and_save(input) =
  input
  |> validate
  !> save_data
  !> notify_user
  !? {
    success(r) => "Complete" !> io.say
    failure(e) => concat("Error: ", e) !> io.say
  };
```

---

## Next Steps (Post-MVP)

- [ ] CLI (`calcium run file.ca`)
- [ ] REPL (Interactive execution environment)
- [ ] More detailed error messages (line numbers, column numbers)
- [ ] Debugger
- [ ] Optimization passes (constant folding, dead code elimination)

---

## References

- [Writing An Interpreter In Go](https://interpreterbook.com/)
- [Writing A Compiler In Go](https://compilerbook.com/)
- [Crafting Interpreters](https://craftinginterpreters.com/)
