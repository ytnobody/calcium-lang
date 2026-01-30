# Calcium-lang

[![CI](https://github.com/ytnobody/calcium-lang/actions/workflows/ci.yml/badge.svg)](https://github.com/ytnobody/calcium-lang/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/ytnobody/calcium-lang/branch/main/graph/badge.svg)](https://codecov.io/gh/ytnobody/calcium-lang)
[![Go Report Card](https://goreportcard.com/badge/github.com/ytnobody/calcium-lang)](https://goreportcard.com/report/github.com/ytnobody/calcium-lang)

> **Note:** This project is currently under active development. APIs and syntax may change without notice.

A functional programming language with pipelines, pattern matching, and effect handling.

**Documentation:** [https://ytnobody.github.io/calcium-pages](https://ytnobody.github.io/calcium-pages)

## Features

- **Pipeline Operator** (`|>`) - Chain function calls in a readable left-to-right style
- **Effect Pipeline** (`!>`) - Chain side-effecting functions with automatic success/failure wrapping
- **Pattern Matching** - Powerful `match` expressions for control flow
- **First-class Functions** - Lambda expressions and closures with proper variable capture
- **Partial Application** - Built-in support for curried functions like `map`, `filter`, `reduce`
- **Array Destructuring** - Unpack arrays into variables with `[a, b, c] = arr` or `[head | tail] = arr`
- **Hashes** - Associative arrays with dot and bracket access
- **String Interpolation** - Embed expressions in strings with `"Hello, ${name}!"`
- **Constraints** - Define and enforce value constraints with `constraint` and `?` operator
- **Chained Comparisons** - Write `0 <= x <= 100` instead of `x >= 0 && x <= 100`
- **Effect Functions** - Distinguish pure functions from side-effecting ones with `func!`
- **Result Types** - Built-in `success(value)` and `failure(error)` for error handling
- **Module System** - Organize code with `use` statements, including GitHub imports
- **Standard Library** - Math, string, array, regex, TOML, HTTP modules

## Installation

### Download Binary (Recommended)

Download pre-built binaries from [GitHub Releases](https://github.com/ytnobody/calcium-lang/releases).

**Supported platforms:**
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64, arm64)

Example for Linux (amd64):
```bash
# Download and extract
curl -LO https://github.com/ytnobody/calcium-lang/releases/latest/download/calcium_Linux_x86_64.tar.gz
tar xzf calcium_Linux_x86_64.tar.gz

# Move to PATH
sudo mv calcium /usr/local/bin/
sudo mv bone /usr/local/bin/
```

### Build from Source

Requires Go 1.24 or later.

```bash
go build -o calcium ./cmd/calcium
go build -o bone ./cmd/bone
```

## Usage

```bash
# Run a Calcium program
calcium hello.ca
calcium run examples/pipeline.ca

# Compile to bytecode
calcium compile program.ca -o program.bone

# Run compiled bytecode
calcium run program.bone

# Run tests
calcium test ./tests     # Run all .test.ca files in directory

# Start interactive REPL
calcium repl

# Show version
calcium version
```

## Quick Start

### Hello World

```calcium
use core.io!

"Hello, Calcium!" !> io.println;
```

### Pipeline Operator

The pipeline operator `|>` passes the left-hand value as the first argument to the right-hand function:

```calcium
use core.io!

// These are equivalent:
io.println("Hello");
"Hello" !> io.println;

// Chain multiple operations
[1, 2, 3, 4, 5]
    |> filter(x => x % 2 == 1)
    |> map(x => x * x)
    |> reduce((a, b) => a + b, 0)
    !> io.println;  // Output: 35
```

### Functions

```calcium
// Named function
func double(x) = x * 2;

// Lambda expression
triple = x => x * 3;

// Multi-parameter lambda
add = (a, b) => a + b;

// Recursive function
func factorial(n) = match n
    0 => 1
    _ => n * factorial(n - 1);
```

### Strings

Calcium supports three string syntaxes:

```calcium
// Double-quoted strings (with interpolation)
name = "World";
"Hello, ${name}!"              // Hello, World!

// Single-quoted strings (simpler escaping for JSON, etc.)
json = '{"name": "Alice", "age": 30}';

// Heredoc (triple-quoted) for multi-line strings
html = """
<html>
    <body>Hello, World!</body>
</html>
""";
```

### String Interpolation

Embed expressions in double-quoted strings using `${...}`:

```calcium
use core.io!

name = "World";
"Hello, ${name}!" !> io.println;  // Hello, World!

x = 10;
y = 20;
"${x} + ${y} = ${x + y}" !> io.println;  // 10 + 20 = 30

arr = [1, 2, 3];
"Array: ${arr}, length: ${len(arr)}" !> io.println;
// Array: [1, 2, 3], length: 3
```

Note: Single-quoted strings and heredocs do not support interpolation.

### Effect Functions

Effect functions (`func!`) perform side effects and automatically wrap their return value in `success()`:

```calcium
use core.io!
use core.http!

// Effect function definition
func! greet(name) = io.println("Hello, " + name);

// Using effect pipeline
"world" !> greet;  // Returns success(null)

// HTTP requests are effects
result = http.get("https://api.example.com/data", {});
// Returns success({status, headers, body, ok}) or failure(error)
```

### Pattern Matching

```calcium
func fizz(n) = match n % 3  0 => "Fizz"  _ => "";
func buzz(n) = match n % 5  0 => "Buzz"  _ => "";

func fizzbuzz_or(result, n) = match result
    "" => to_string(n)
    _ => result;

func fizzbuzz(n) = fizz(n) + buzz(n) |> fizzbuzz_or(n);

func fib(n) = match n
    0 => 0
    1 => 1
    _ => fib(n - 1) + fib(n - 2);
```

### Partial Application

`map`, `filter`, and `reduce` work seamlessly with pipelines using `x |> f(y)` = `f(x, y)`:

```calcium
// Direct call: map(arr, fn)
map([1, 2, 3], x => x * 2);  // [2, 4, 6]

// Pipeline: arr |> map(fn) becomes map(arr, fn)
[1, 2, 3, 4, 5]
    |> filter(x => x > 2)
    |> map(x => x * 10)
    !> io.println;  // [30, 40, 50]
```

### Closures

Functions capture variables from their enclosing scope:

```calcium
func make_adder(n) = x => x + n;

add5 = make_adder(5);
add5(10);  // 15
```

### Arrays

```calcium
use core.array

numbers = [1, 2, 3, 4, 5];

// Built-in functions
map(numbers, x => x * 2);            // [2, 4, 6, 8, 10]
filter(numbers, x => x > 2);         // [3, 4, 5]
reduce(numbers, (a, b) => a + b, 0); // 15

// Array module functions
numbers |> array.reverse;   // [5, 4, 3, 2, 1]
numbers |> array.sum;       // 15
array.take(numbers, 3);     // [1, 2, 3]
array.unique([1,2,2,3,1]);  // [1, 2, 3]

// Generate ranges
range(1, 6);  // [1, 2, 3, 4, 5]
```

### Array Destructuring

```calcium
// Unpack array elements into variables
[a, b, c] = [10, 20, 30];
// a = 10, b = 20, c = 30

// Head and tail pattern (like Haskell/Elixir)
[first | rest] = [1, 2, 3, 4, 5];
// first = 1, rest = [2, 3, 4, 5]

// Useful for recursive functions
func sum_list(arr) = match len(arr)
    0 => 0
    _ => head(arr) + sum_list(tail(arr));
```

### Hashes (Associative Arrays)

```calcium
// Hash literal with identifier keys
person = {name: "Alice", age: 30, city: "Tokyo"};

// Dot access
person.name;       // "Alice"
person.age;        // 30

// Bracket access (supports dynamic keys)
person["city"];    // "Tokyo"
key = "name";
person[key];       // "Alice"

// Nested hashes
user = {
    profile: {name: "Bob", age: 25},
    active: true
};
user.profile.name;  // "Bob"

// Built-in hash functions
keys(person);      // ["name", "age", "city"]
values(person);    // ["Alice", 30, "Tokyo"]
len(person);       // 3
```

### Constraints

Constraints define validation rules that can be checked at runtime:

```calcium
// Define a constraint
constraint Positive(n) = n > 0;
constraint InRange(n) = 0 <= n <= 100;  // Chained comparisons supported

// Check constraint with pipe: returns success(value) or failure(value)
10 |> Positive?;   // success(10)
-5 |> Positive?;   // failure(-5)

// Use constraints in function parameters
func safe_divide(x, y: Positive?) = x / y;

safe_divide(10, 2);   // success(5)
safe_divide(10, 0);   // failure(0)
safe_divide(10, -1);  // failure(-1)
```

### Regular Expressions

```calcium
use core.regex

// Test if pattern matches
regex.matches("hello world", /world/);  // true

// Find matches
regex.find("hello123world", /\d+/);     // success("123")
regex.find_all("a1b2c3", /\d/);         // ["1", "2", "3"]

// Replace
regex.replace("hello world", /world/, "calcium");  // "hello calcium"

// Split
regex.split("a1b2c3", /\d/);            // ["a", "b", "c"]

// Capture groups
regex.capture("2024-01-15", /(\d+)-(\d+)-(\d+)/);
// success(["2024-01-15", "2024", "01", "15"])
```

### HTTP Client

```calcium
use core.http!
use core.io!

// GET request
result = http.get("https://api.example.com/users", {});
io.println(result);
// success({status: 200, headers: {...}, body: "...", ok: true})

// POST request with JSON
result = http.post_json("https://api.example.com/users", "{\"name\": \"Alice\"}");

// Custom request
result = http.request({
    method: "GET",
    url: "https://api.example.com/data",
    headers: {"Authorization": "Bearer token123"}
});
```

### TOML Parsing

```calcium
use core.toml
use core.io!

// Parse TOML string
toml_str = "[package]
name = 'my-app'
version = '1.0.0'
";

result = toml.parse(toml_str);
io.println(result);
// success({package: {name: "my-app", version: "1.0.0"}})

// Convert hash to TOML
data = {name: "test", value: 123};
toml.stringify(data);
// success("name = \"test\"\nvalue = 123\n")
```

### External Modules

Import modules from the Boneyard registry or GitHub:

```calcium
// Import from Boneyard registry (recommended)
use ytnobody/json;
use ytnobody/json!;  // Effect module

// Import from GitHub URL
use "github.com/ytnobody/json-calcium";
```

#### Installing Modules with bone

Use the `bone` package manager to install modules:

```bash
# Install to project (calcium_modules/)
bone add ytnobody/json

# Install to global cache (~/.calcium/cache/)
bone add --global ytnobody/json

# Install a specific version
bone add ytnobody/json@1.0.0
```

Module resolution order:
1. In-memory cache
2. Local `calcium_modules/author/module/`
3. Global cache `~/.calcium/cache/author/module/`
4. Auto-fetch from GitHub (saved to global cache)

## Standard Library

### core.io!

| Function | Description |
|----------|-------------|
| `io.println(value)` | Print value with newline |
| `io.print(value)` | Print value without newline |
| `io.read_file(path)` | Read file contents |
| `io.write_file(path, content)` | Write content to file |

### core.math

| Function | Description |
|----------|-------------|
| `math.pi` | Pi constant (3.14159...) |
| `math.e` | Euler's number (2.71828...) |
| `math.abs(n)` | Absolute value |
| `math.floor(n)` | Round down |
| `math.ceil(n)` | Round up |
| `math.round(n)` | Round to nearest integer |
| `math.sqrt(n)` | Square root |
| `math.pow(base, exp)` | Exponentiation |
| `math.min(a, b)` | Minimum of two values |
| `math.max(a, b)` | Maximum of two values |
| `math.clamp(n, min, max)` | Clamp value to range |
| `math.sin(n)`, `math.cos(n)`, `math.tan(n)` | Trigonometric functions |

### core.string

| Function | Description |
|----------|-------------|
| `string.length(s)` | String length |
| `string.trim(s)` | Remove leading/trailing whitespace |
| `string.upper(s)` | Convert to uppercase |
| `string.lower(s)` | Convert to lowercase |
| `string.split(s, sep)` | Split string by separator |
| `string.join(arr, sep)` | Join array with separator |
| `string.contains(s, sub)` | Check if contains substring |
| `string.starts_with(s, prefix)` | Check prefix |
| `string.ends_with(s, suffix)` | Check suffix |
| `string.replace(s, old, new)` | Replace all occurrences |
| `string.substring(s, start, end)` | Extract substring |
| `string.index_of(s, sub)` | Find substring position |
| `string.char_at(s, index)` | Get character at index |
| `string.repeat(s, n)` | Repeat string n times |
| `string.pad_left(s, len, char)` | Pad on left |
| `string.pad_right(s, len, char)` | Pad on right |

### core.array

| Function | Description |
|----------|-------------|
| `array.reverse(arr)` | Reverse an array |
| `array.sum(arr)` | Sum all elements |
| `array.product(arr)` | Product of all elements |
| `array.take(arr, n)` | Take first n elements |
| `array.drop(arr, n)` | Drop first n elements |
| `array.slice(arr, start, end)` | Extract portion |
| `array.flatten(arr)` | Flatten nested array |
| `array.unique(arr)` | Remove duplicates |
| `array.zip(arr1, arr2)` | Combine two arrays into pairs |
| `array.index_of(arr, elem)` | Find element position |
| `array.find(arr, pred)` | Find first matching element |
| `array.any(arr, pred)` | Check if any element matches |
| `array.all(arr, pred)` | Check if all elements match |
| `array.count(arr, pred)` | Count matching elements |
| `array.partition(arr, pred)` | Split by predicate |
| `array.chunk(arr, n)` | Split into chunks of size n |
| `array.sort(arr)` | Sort array in ascending order |
| `array.sort_by(arr, cmp)` | Sort with custom comparison function |

### core.regex

| Function | Description |
|----------|-------------|
| `regex.matches(s, pattern)` | Test if pattern matches |
| `regex.find(s, pattern)` | Find first match |
| `regex.find_all(s, pattern)` | Find all matches |
| `regex.replace(s, pattern, replacement)` | Replace all matches |
| `regex.replace_first(s, pattern, replacement)` | Replace first match only |
| `regex.split(s, pattern)` | Split by pattern |
| `regex.capture(s, pattern)` | Extract capture groups |

### core.toml

| Function | Description |
|----------|-------------|
| `toml.parse(s)` | Parse TOML string to hash |
| `toml.stringify(hash)` | Convert hash to TOML string |

### core.http!

| Function | Description |
|----------|-------------|
| `http.get(url, headers)` | GET request |
| `http.post(url, body, content_type, headers)` | POST request |
| `http.put(url, body, content_type, headers)` | PUT request |
| `http.del(url, headers)` | DELETE request |
| `http.request(options)` | Custom request |
| `http.post_json(url, data)` | POST with JSON content type |
| `http.post_form(url, data)` | POST with form content type |

### Built-in Functions

| Function | Description |
|----------|-------------|
| `map(arr, fn)` | Apply function to each element |
| `filter(arr, pred)` | Keep elements matching predicate |
| `reduce(arr, fn, init)` | Fold array to single value |
| `range(start, end)` | Generate array of integers |
| `len(x)` | Get length of array/string/hash |
| `concat(a, b)` | Concatenate arrays or strings |
| `to_string(value)` | Convert to string |
| `head(arr)` | Get first element |
| `tail(arr)` | Get all but first element |
| `push(arr, elem)` | Append element to array |
| `get(collection, key)` | Get element by key/index |
| `has(hash, key)` | Check if hash has key |
| `keys(hash)` | Get all keys from hash |
| `values(hash)` | Get all values from hash |
| `success(value)` | Wrap value in success |
| `failure(error)` | Wrap error in failure |

## Package Manager (bone)

`bone` is the package manager for Calcium. It manages dependencies and integrates with the Boneyard module registry.

> **Note:** When you download Calcium from [GitHub Releases](https://github.com/ytnobody/calcium-lang/releases), `bone` is included in the archive.

### Usage

```bash
# Initialize a new project
bone init my-project

# Add a module (installs to calcium_modules/)
bone add ytnobody/json

# Add to global cache
bone add --global ytnobody/json

# List installed modules
bone list

# Update modules
bone update

# Remove a module
bone remove ytnobody/json

# Show configuration
bone config

# Set custom registry URL
bone config set registry_url https://example.com/registry
```

### Project Structure

After `bone init`, your project looks like:

```
my-project/
├── meta.toml           # Project metadata
├── calcium.lock        # Lock file (versions)
├── calcium_modules/    # Local modules
│   └── author/
│       └── module/
│           └── mod.ca
└── main.ca             # Entry point
```

### meta.toml

```toml
name = "my-project"
author = "yourname"
description = "My Calcium project"
license = "MIT"
entry = "main.ca"

[dependencies]
"ytnobody/json" = "^1.0.0"
```

### Configuration

Configuration is stored in `~/.calcium/config.toml`:

```toml
registry_url = "https://raw.githubusercontent.com/ytnobody/boneyard/main/index"
```

## Boneyard Registry

[Boneyard](https://github.com/ytnobody/boneyard) is the official module registry for Calcium.

### Publishing a Module

1. Create `meta.toml` in your repository:

```toml
name = "my-module"
author = "YOURNAME"
description = "My awesome module"
license = "MIT"
keywords = ["utility"]
entry = "mod.ca"
```

2. (Optional) Create a release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

3. Submit your module by opening an issue at [Boneyard](https://github.com/ytnobody/boneyard/issues/new) with:

```
https://raw.githubusercontent.com/YOURNAME/my-module/main/meta.toml
```

### Using Modules

```calcium
use core.io!;
use ytnobody/json;

data = {name: "Calcium", version: 1};
json_str = json.stringify(data);
io.println(json_str);  // {"name":"Calcium","version":1}

result = json.parse('{"hello": "world"}');
parsed = result !? { success(v) => v  failure(e) => {} };
io.println(parsed["hello"]);  // world
```

## Examples

See the `examples/` directory:

- `hello.ca` - Hello World
- `factorial.ca` - Factorial with pattern matching
- `fibonacci.ca` - Fibonacci sequence
- `pipeline.ca` - Pipeline operator demonstrations
- `strings.ca` - String manipulation
- `hash.ca` - Hash (associative array) operations
- `constraint.ca` - Constraint validation examples
- `regex_demo.ca` - Regular expression examples

Run an example:

```bash
calcium examples/pipeline.ca
```

## Compiler Optimization

Calcium includes an optimizer with multiple optimization levels:

| Level | Flag | Description |
|-------|------|-------------|
| O0 | `-O0` | No optimization (fastest compilation) |
| O1 | `-O1` | AST optimizations (default) |
| O2 | `-O2` | AST + bytecode optimizations |

### Optimizations

- **Constant Folding** - Evaluates constant expressions at compile time
- **Dead Code Elimination** - Removes unreachable code paths
- **Common Subexpression Elimination** - Reuses identical pure expressions
- **Peephole Optimization** (O2) - Bytecode-level optimizations

```bash
# Compile with maximum optimization
calcium compile -O2 program.ca -o program.bone
```

## Project Structure

```
calcium/
├── cmd/
│   ├── calcium/       # Calcium CLI
│   └── bone/          # Package manager CLI
├── pkg/
│   ├── ast/           # Abstract Syntax Tree
│   ├── bone/          # Package manager core
│   ├── bytecode/      # Bytecode definitions
│   ├── compiler/      # Compiler (AST to bytecode)
│   ├── lexer/         # Lexical analyzer
│   ├── optimizer/     # Optimization passes
│   ├── parser/        # Parser
│   ├── token/         # Token definitions
│   ├── value/         # Runtime value types
│   └── vm/            # Virtual machine
│       └── stdlib/    # Standard library (Calcium sources)
├── examples/          # Example programs
└── specs/             # Design specifications
```

## License

MIT
