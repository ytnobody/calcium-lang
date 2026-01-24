# Calcium Language Reference Manual

This document provides a complete reference for the Calcium programming language.

## Table of Contents

1. [Lexical Structure](#lexical-structure)
2. [Values and Types](#values-and-types)
3. [Variables](#variables)
4. [Operators](#operators)
5. [Control Flow](#control-flow)
6. [Functions](#functions)
7. [Constraints](#constraints)
8. [Modules](#modules)
9. [Asynchronous Programming](#asynchronous-programming)
10. [Standard Library](#standard-library)
11. [Built-in Functions](#built-in-functions)
12. [Testing](#testing)
13. [Compiler Optimization](#compiler-optimization)

---

## Lexical Structure

### Comments

```calcium
// Single line comment

/*
  Multi-line comment
  Can span multiple lines
*/
```

### Statement Terminator

Statements end with a semicolon `;`. Line breaks are treated as whitespace, allowing expressions to span multiple lines.

```calcium
x = 5;
name = "calcium";

// Multi-line expression
result = data
    |> transform
    |> validate;
```

### Identifiers

Identifiers start with a letter or underscore, followed by letters, digits, or underscores.

```calcium
foo
_bar
myVariable123
```

### Keywords

```
func func! constraint namespace use match
map filter reduce in has keys values hash len
success failure return true false
```

---

## Values and Types

Calcium has the following value types:

| Type | Examples | Description |
|------|----------|-------------|
| Boolean | `true`, `false` | Logical values |
| Integer | `42`, `-10`, `0xFF`, `0b1010` | Whole numbers |
| Float | `3.14`, `-0.5`, `1.5e10` | Decimal numbers |
| String | `"hello"`, `"世界"` | Text values |
| Array | `[1, 2, 3]` | Ordered collections |
| Hash | `{name: "Alice", age: 30}` | Key-value mappings |
| Function | `(x) => x * 2` | Callable values |
| Result | `success(v)`, `failure(e)` | Success/failure wrapper |

### Numbers

```calcium
// Integers
42
-10
0xFF        // Hexadecimal (255)
0b1010      // Binary (10)
1_000_000   // With separators

// Floats
3.14
-0.5
1.5e10      // Scientific notation
2.5e-3
```

### Strings

Calcium supports three string syntaxes:

#### Double-quoted strings

```calcium
"Hello, World!"
"Line 1\nLine 2"    // Newline
"Tab:\tHere"        // Tab
"Quote: \"text\""   // Escaped quote
"Backslash: \\"     // Escaped backslash
```

Escape sequences: `\\`, `\"`, `\n`, `\t`, `\r`

Double-quoted strings support interpolation with `${...}`:

```calcium
name = "Alice";
"Hello, ${name}!";           // "Hello, Alice!"
"Sum: ${1 + 2}";             // "Sum: 3"
```

#### Single-quoted strings

Single-quoted strings are useful when the content contains double quotes:

```calcium
'Hello, World!'
'Say "hello"'               // No escaping needed for double quotes
'It\'s easy'                // Escape single quotes with \'
```

Escape sequences: `\\`, `\'`, `\n`, `\t`, `\r`

Note: Single-quoted strings do not support interpolation.

#### Heredoc (triple-quoted strings)

Heredocs allow multi-line strings without escaping:

```calcium
text = """
This is a multi-line string.
No escaping needed for "quotes" or 'apostrophes'.
Newlines are preserved.
""";

// Useful for JSON, HTML, SQL, etc.
json = """
{
    "name": "Alice",
    "age": 30
}
""";
```

- Leading newline after opening `"""` is stripped
- Trailing newline before closing `"""` is stripped
- Content is taken literally (no escape processing)
- No interpolation support

### Arrays

```calcium
numbers = [1, 2, 3, 4, 5];
mixed = [1, "hello", [2, 3]];

// Index access (0-based)
numbers[0];     // 1
numbers[-1];    // 5 (last element)
numbers[-2];    // 4 (second from last)

// Array concatenation with space separator
[1 2 3]              // [1, 2, 3]
[[1, 2] [3, 4]]      // [1, 2, 3, 4] (flattened)

// Comma separator preserves nesting
[[1, 2], [3, 4]]     // [[1, 2], [3, 4]]
```

### Array Destructuring

```calcium
// Simple destructuring
[a, b, c] = [10, 20, 30];
// a = 10, b = 20, c = 30

// Head and tail pattern
[first | rest] = [1, 2, 3, 4, 5];
// first = 1, rest = [2, 3, 4, 5]

// Multiple elements before rest
[a, b | tail] = [1, 2, 3, 4, 5];
// a = 1, b = 2, tail = [3, 4, 5]
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

// String keys with special characters
data = {"my-key": 42, "another key": 100};
data["my-key"];    // 42

// Nested hashes
user = {
    profile: {name: "Bob", age: 25},
    active: true
};
user.profile.name;  // "Bob"
```

### Result Type

The result type represents success or failure:

```calcium
success(42);         // Success with value 42
failure("error");    // Failure with error message

// Pattern matching on results
result !? {
    success(v) => v
    failure(e) => handle_error(e)
};
```

---

## Variables

Variables bind names to values. Reassignment is not allowed.

```calcium
x = 5;
name = "calcium";
items = [1, 2, 3];
double = (x) => x * 2;

// Reassignment is an error
x = 10;  // Error!
```

---

## Operators

### Arithmetic Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+` | Addition | `5 + 3` → `8` |
| `-` | Subtraction | `5 - 3` → `2` |
| `*` | Multiplication | `5 * 3` → `15` |
| `/` | Division | `5 / 2` → `2.5` |
| `%` | Modulo | `5 % 2` → `1` |
| `**` | Exponentiation | `2 ** 3` → `8` |

Note: `+` is for numbers only. Use `concat()` for string concatenation.

### Comparison Operators

| Operator | Description |
|----------|-------------|
| `==` | Equal |
| `!=` | Not equal |
| `<` | Less than |
| `>` | Greater than |
| `<=` | Less than or equal |
| `>=` | Greater than or equal |

Chained comparisons are supported:

```calcium
0 <= n <= 100    // Equivalent to: 0 <= n && n <= 100
```

### Logical Operators

| Operator | Description |
|----------|-------------|
| `&&` | Logical AND (short-circuit) |
| `\|\|` | Logical OR (short-circuit) |
| `!` | Logical NOT |

### Pipeline Operators

| Operator | Description |
|----------|-------------|
| `\|>` | Pure pipeline |
| `!>` | Effect pipeline (in `func!` only) |
| `!?` | Result handler |

```calcium
// Pure pipeline
[1, 2, 3] |> map(x => x * 2) |> filter(x => x > 2);

// Effect pipeline with error handling
data !> save !> notify !? {
    success(v) => v
    failure(e) => handle(e)
};
```

### Spread Operator

The `...` operator expands arrays into argument lists:

```calcium
func add(x, y) = x + y;

[2, 3]... |> add;       // add(2, 3) → 5
add([2, 3]...);         // add(2, 3) → 5

pair = [10, 20];
pair... |> add;         // 30
```

### Operator Precedence

From highest to lowest:

| Precedence | Operators |
|------------|-----------|
| 1 | `f(x)`, `obj.key`, `...` |
| 2 | `-` (unary), `!` |
| 3 | `**` |
| 4 | `*`, `/`, `%` |
| 5 | `+`, `-` |
| 6 | `<`, `>`, `<=`, `>=` |
| 7 | `==`, `!=` |
| 8 | `&&` |
| 9 | `\|\|` |
| 10 | `\|>`, `!>` |
| 11 | `=` |

---

## Control Flow

### match Expression

Pattern matching for control flow:

```calcium
// Match on value
func describe(x) = match x
    0 => "zero"
    1 => "one"
    _ => "other";

// Match with conditions
func fizzbuzz(n) = match true
    n % 15 == 0 => "FizzBuzz"
    n % 3 == 0 => "Fizz"
    n % 5 == 0 => "Buzz"
    true => to_string(n);

// Match on result type
result !? {
    success(v) => process(v)
    failure(e) => handle(e)
};
```

The wildcard `_` matches anything and ensures exhaustiveness.

---

## Functions

### Pure Functions

Functions without side effects use `func`:

```calcium
func add(a, b) = a + b;

func double(x) = x * 2;

func factorial(n) = match n
    0 => 1
    _ => n * factorial(n - 1);
```

### Effect Functions

Functions with side effects use `func!`:

```calcium
use core.io!;

func! greet(name) = io.println("Hello, " + name);

func! save(data) = ...;
```

### Lambda Expressions

Anonymous functions:

```calcium
// Single parameter (no parentheses needed)
double = x => x * 2;

// Multiple parameters
add = (a, b) => a + b;

// In pipelines
[1, 2, 3] |> map(x => x * 2);
```

### Partial Application

`map`, `filter`, and `reduce` support partial application:

```calcium
// Full application
map(x => x * 2, [1, 2, 3]);  // [2, 4, 6]

// Partial application returns a function
double_all = map(x => x * 2);
double_all([1, 2, 3]);        // [2, 4, 6]

// Perfect for pipelines
[1, 2, 3] |> filter(x => x > 1) |> map(x => x * 10);
```

### Closures

Functions capture variables from their enclosing scope:

```calcium
func make_adder(n) = x => x + n;

add5 = make_adder(5);
add5(10);  // 15
```

### Rest Parameters

```calcium
func sum(| items) = items |> reduce((a, b) => a + b, 0);

sum(1, 2, 3, 4, 5);  // 15

func process(first, second | rest) = ...;
```

---

## Constraints

Constraints define validation rules for values:

### Defining Constraints

```calcium
constraint Positive(n) = n > 0;
constraint Age(n) = 0 <= n <= 150;
constraint NonEmpty(s) = len(s) > 0;
constraint Status(s) = s == "active" || s == "pending" || s == "closed";
```

### Checking Constraints

Use `?` to check a constraint:

```calcium
10 |> Positive?;   // true
-5 |> Positive?;   // false
25 |> Age?;        // true
```

### Constraints in Functions

```calcium
func divide(a, b: NonZero?) = a / b;

func register(name: NonEmpty?, age: Age?) = ...;
```

### Result Handling with `!?`

```calcium
42 |> Positive? !? {
    success(v) => "Valid: " + to_string(v)
    failure(v) => "Invalid: " + to_string(v)
};
```

---

## Modules

### Using Modules

```calcium
use core.io!;      // Effect module (with !)
use core.math;     // Pure module

io.println("Hello");
math.sqrt(16);
```

### Defining Namespaces

```calcium
// math.ca
namespace math;

func add(a, b) = a + b;
func multiply(a, b) = a * b;
```

### Importing Specific Items

```calcium
use math { add, multiply };

add(2, 3);  // Can use directly
```

---

## Asynchronous Programming

### Overview

Calcium provides async programming through the `core.async!` and `core.schedule!` modules.

### async.stay

Creates an event loop with state:

```calcium
use core.async!;
use core.schedule!;

result = async.stay(count: 0) {
    src = schedule.timeout(100);
    handler = async.expects((ev) => async.leave("done"), src);
    handler.ready()
};
```

### async.spawn

Spawns a background task:

```calcium
task = async.spawn(() => 42);

// Task properties
task.status;   // "pending", "running", "completed", "failed", "cancelled"
task.result;   // Result value (when completed)
task.done;     // Event source for completion
```

### async.expects

Creates an event handler:

```calcium
handler = async.expects((event) => {
    // Handle event
    async.leave(result)
}, eventSource);

handler.ready();   // Activate handler
handler.pause();   // Pause handler
handler.resume();  // Resume handler
handler.status;    // "dormant", "active", "paused", "cancelled"
```

### async.continue / async.leave

```calcium
async.continue({count: count + 1});  // Update state, continue loop
async.leave(result);                  // Exit loop with result
```

### async.cancel

```calcium
async.cancel(handler);  // Cancel a handler
async.cancel(task);     // Cancel a task
```

### async.all

Wait for multiple tasks:

```calcium
tasks = [
    async.spawn(() => 10),
    async.spawn(() => 20),
    async.spawn(() => 30)
];

results = async.all(tasks);  // [10, 20, 30]
```

### schedule.timeout

One-time timer:

```calcium
use core.schedule!;

src = schedule.timeout(1000);  // Fires after 1000ms
```

### schedule.interval

Repeating timer:

```calcium
src = schedule.interval(100);  // Fires every 100ms
```

---

## Standard Library

### core.io!

| Function | Description |
|----------|-------------|
| `io.print(value)` | Print without newline |
| `io.println(value)` | Print with newline |
| `io.say(value)` | Alias for println |
| `io.read_file(path)` | Read file contents |
| `io.write_file(path, content)` | Write to file |
| `io.format(template, ...)` | Format string |
| `io.stdin` | Event source for stdin lines |
| `io.eof` | Event source for EOF |

### core.math

| Function | Description |
|----------|-------------|
| `math.abs(n)` | Absolute value |
| `math.min(a, b)` | Minimum value |
| `math.max(a, b)` | Maximum value |
| `math.floor(n)` | Round down |
| `math.ceil(n)` | Round up |
| `math.round(n)` | Round to nearest |
| `math.sqrt(n)` | Square root |
| `math.pow(base, exp)` | Exponentiation |

### core.string

| Function | Description |
|----------|-------------|
| `string.trim(s)` | Remove whitespace |
| `string.upper(s)` | Uppercase |
| `string.lower(s)` | Lowercase |
| `string.split(s, sep)` | Split string |
| `string.join(arr, sep)` | Join array |
| `string.contains(s, sub)` | Check substring |
| `string.starts_with(s, prefix)` | Check prefix |
| `string.ends_with(s, suffix)` | Check suffix |
| `string.replace(s, old, new)` | Replace text |
| `string.substring(s, start, end)` | Extract substring |
| `string.char_at(s, index)` | Get character |
| `string.index_of(s, sub)` | Find position |

### core.array

| Function | Description |
|----------|-------------|
| `array.reverse(arr)` | Reverse array |
| `array.sum(arr)` | Sum elements |
| `array.product(arr)` | Product of elements |
| `array.take(arr, n)` | First n elements |
| `array.drop(arr, n)` | Remove first n elements |

### core.assert!

Testing and assertion module. All functions require a label as the first argument.

| Function | Description |
|----------|-------------|
| `assert.eq(label, actual, expected)` | Assert equality |
| `assert.neq(label, actual, expected)` | Assert inequality |
| `assert.is_true(label, value)` | Assert value is true |
| `assert.is_false(label, value)` | Assert value is false |
| `assert.is_null(label, value)` | Assert value is null |
| `assert.is_type(label, value, type)` | Assert value has type |
| `assert.len_eq(label, collection, length)` | Assert collection length |
| `assert.throws(label, result)` | Assert result is failure |
| `assert.fail(message)` | Force test failure |
| `assert.section(name)` | Print section header |

Example:

```calcium
use core.assert!;

assert.section("Math tests");
assert.eq("addition", 1 + 1, 2);
assert.is_true("comparison", 5 > 3);
```

### core.async!

| Function | Description |
|----------|-------------|
| `async.stay(state) { ... }` | Event loop |
| `async.spawn(fn)` | Spawn task |
| `async.expects(fn, src)` | Create handler |
| `async.continue(state)` | Update state |
| `async.leave(value)` | Exit loop |
| `async.cancel(target)` | Cancel task/handler |
| `async.all(tasks)` | Wait for all tasks |

### core.schedule!

| Function | Description |
|----------|-------------|
| `schedule.timeout(ms)` | One-time timer |
| `schedule.interval(ms)` | Repeating timer |

---

## Built-in Functions

These functions are available without imports:

| Function | Description |
|----------|-------------|
| `len(x)` | Length of array or string |
| `concat(...)` | Concatenate strings |
| `to_string(x)` | Convert to string |
| `get(coll, key)` | Safe access → success/failure |
| `get(coll, key, default)` | Safe access with default |
| `has(coll, key)` | Check if key exists |
| `head(arr)` | First element |
| `tail(arr)` | All but first element |
| `push(arr, elem)` | Append element |
| `range(start, end)` | Generate integer array |
| `keys(hash)` | Get all keys |
| `values(hash)` | Get all values |
| `map(fn, arr)` | Transform elements |
| `filter(pred, arr)` | Filter elements |
| `reduce(fn, init, arr)` | Fold array |

---

## Testing

### Test Files

Test files use the `.test.ca` extension. They are regular Calcium source files that use the `core.assert!` module.

```calcium
// math.test.ca
use core.io!;
use core.assert!;

io.println("Math Tests");
io.println("==========");

assert.section("Addition");
assert.eq("1 + 1", 1 + 1, 2);
assert.eq("negative", -5 + 3, -2);

assert.section("Multiplication");
assert.eq("2 * 3", 2 * 3, 6);

io.println("==========");
io.println("Tests complete!");
```

### Running Tests

Use the `calcium test` command to run all test files in a directory:

```bash
# Run all .test.ca files in ./tests
calcium test ./tests

# Run tests in current directory
calcium test .
```

The test runner will:
- Find all `.test.ca` files in the specified directory
- Execute each test file
- Display pass/fail summary

---

## Compiler Optimization

Calcium includes an optimizer that performs various transformations to improve code efficiency.

### Optimization Levels

| Level | Description |
|-------|-------------|
| O0 | No optimization - fastest compilation |
| O1 | AST optimizations (default) |
| O2 | AST + bytecode optimizations |

### AST Optimizations (O1+)

#### Constant Folding

Evaluates constant expressions at compile time:

```calcium
x = 2 + 3;        // Optimized to: x = 5;
y = "hello" + " world";  // Optimized to: y = "hello world";
```

#### Dead Code Elimination

Removes unreachable code:

```calcium
func example(x) = match x
    true => "yes"
    false => "no"
    _ => "unreachable";  // This branch is eliminated
```

#### Common Subexpression Elimination (CSE)

Detects and optimizes duplicate pure expressions:

```calcium
// The compiler detects that expensive(x) is called twice
// and can optimize to compute it only once
result = use_both(expensive(x), expensive(x));
```

**Note:** CSE only applies to pure expressions. Effect functions (`func!`) are never deduplicated because their side effects must occur each time.

### Bytecode Optimizations (O2)

#### Peephole Optimization

Low-level bytecode transformations:

- Redundant load/store elimination
- Jump chain optimization
- Constant propagation

### Usage

```bash
# Default optimization (O1)
calcium run program.ca

# No optimization
calcium run -O0 program.ca

# Maximum optimization
calcium compile -O2 program.ca -o program.bone
```

---

## File Extension

Calcium source files use `.ca` or `.calcium` extension.

---

## Entry Point

Top-level code is executed directly. No `main` function is required.

```calcium
use core.io!;

x = 10;
y = 20;
io.println(x + y);  // 30
```

Command-line arguments are available in `args`:

```calcium
args[0];      // First argument
len(args);    // Number of arguments
```
