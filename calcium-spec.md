# Calcium Language Specification

Calcium is an arithmetic-style programming language designed to be writable by anyone who has learned basic mathematics.

## Design Philosophy

- Use "constraints" instead of types to handle values
- Minimize control structures (no if/else/while)
- Disallow reassignment
- Separate side effects at the syntax level
- Use only concepts that can be explained with mathematical vocabulary

## Comments

Single-line and multi-line comments are supported.

```calcium
// Single-line comment

/*
  Multi-line comment
  Can span multiple lines
*/
```

## Statement Termination

Statements end with a semicolon `;`. Line breaks are treated as whitespace, so expressions can freely span multiple lines.

```calcium
x = 5;
name = "calcium";

// Multi-line expression
result = data
  |> transform
  |> validate;

items = [
  1,
  2,
  3
];

// Function definition
func add(a, b) = a + b;
```

## Values

Calcium can handle the following 6 types of values:

- Booleans: `true`, `false`
- Numbers: `42`, `3.14`, `-10`
  - Integers: `42`, `-10`
  - Floating point: `3.14`, `-0.5`
  - Hexadecimal: `0xFF`, `0x1A3F`
  - Binary: `0b1010`, `0b11110000`
  - Scientific notation: `1.5e10`, `2.5e-3`
  - Numeric separators: `1_000_000`, `0xFF_FF`
- Strings: `"hello"`, `"world"`
  - Escape sequences: `\\`(backslash), `\"`(double quote), `\n`(newline), `\t`(tab), `\r`(carriage return)
- Arrays: `[1, 2, 3]`, `["a", "b", "c"]`
- Functions: `(x) => x * 2`
- Result types: `success(value)`, `failure(error)`

### Result Type (success/failure)

`success` and `failure` are built-in result types.

```calcium
// Create results
success(42);         // Success value
failure("error");    // Failure value

// Pattern match with match
match result
  success(v) => v
  failure(e) => handle_error(e)
```

Effect functions (`func!`) implicitly return result types:
- On success: `success(result)`
- On failure: `failure(error)`

User-defined variant types (algebraic data types) do not exist.

### About null

Calcium has no `null`. Situations where "a value may not exist" are explicitly handled using the `get` function and `success/failure` pattern.

```calcium
data = ["name", "Tanaka"];

// Direct access (assuming key always exists)
data.name;                   // "Tanaka"
data.age;                    // Error (key doesn't exist)

// Safe access
data |> get("name");         // success("Tanaka")
data |> get("age");          // failure("key not found")
data |> get("age", 0);       // 0 (with default value)

// JSON null is treated as failure
json = parse_json('{"name": "Tanaka", "age": null}');
json |> get("name");         // success("Tanaka")
json |> get("age");          // failure("null")
json |> get("age", 0);       // 0
```

### Variables

Variables give names to values. Reassignment is not allowed.

```calcium
x = 5
name = "calcium"
items = [1, 2, 3]
double = (x) => x * 2
```

Reassignment is an error:

```calcium
x = 5
x = 10  // Error
```

### Arrays

Arrays are lists of values.

```calcium
numbers = [1, 2, 3, 4, 5];
mixed = [1, "hello", [2, 3]];
```

Concatenation in array literals (space-separated):

When elements in an array literal are separated by spaces, arrays are concatenated (flattened). When separated by commas, they are kept as individual elements.

```calcium
// Space-separated = concatenation
[1 2 3]              // [1, 2, 3]
[[1, 2] [3, 4]]      // [1, 2, 3, 4]
[[1] [2] [3]]        // [1, 2, 3]

// Comma-separated = individual elements
[1, 2, 3]            // [1, 2, 3]
[[1, 2], [3, 4]]     // [[1, 2], [3, 4]] (nesting preserved)

// Same with variables
a = [1, 2];
b = [3, 4];
[a b]                // [1, 2, 3, 4] (concatenated)
[a, b]               // [[1, 2], [3, 4]] (nested)

// Can mix both
[a [5, 6] b]         // [1, 2, 5, 6, 3, 4]
```

Index access (0-based):

```calcium
numbers = [10, 20, 30];

numbers[0];    // 10
numbers[2];    // 30
numbers[-1];   // 30 (from end)
numbers[-2];   // 20 (second from end)

// Out of bounds access is an error
numbers[10];   // Error

// Use get for safe access
numbers |> get(0);      // success(10)
numbers |> get(10);     // failure("index out of bounds")
numbers |> get(10, 0);  // 0 (default value)
```

Array operations are provided in the `array` namespace:

```calcium
use array;

numbers = [10, 20, 30, 40, 50];

array.concat([1, 2], [3, 4]);    // [1, 2, 3, 4]
array.slice(numbers, 1, 3);      // [20, 30] (index 1 to less than 3)
array.slice(numbers, 2);         // [30, 40, 50] (index 2 to end)
array.first(numbers);            // success(10)
array.last(numbers);             // success(50)
array.reverse(numbers);          // [50, 40, 30, 20, 10]
array.contains(numbers, 30);     // true
array.index_of(numbers, 30);     // success(2)
array.unique([1, 2, 2, 3]);      // [1, 2, 3]
```

`len` is available by default (works for both arrays and strings):

```calcium
len([1, 2, 3]);    // 3
len("hello");      // 5
```

Array destructuring (spread in variable binding):

```calcium
[head | tail] = [1, 2, 3];
// head = 1, tail = [2, 3]

[first, second | rest] = [1, 2, 3, 4, 5];
// first = 1, second = 2, rest = [3, 4, 5]

[a, b, c] = [1, 2, 3];
// a = 1, b = 2, c = 3

// Empty array or insufficient elements causes error
[head | tail] = [];  // Error
[a, b, c] = [1, 2];  // Error
```

### Hashes (Associative Arrays)

Hashes are represented as arrays in the form "key, value, key, value, ...".

```calcium
person = ["name", "Tanaka", "age", 25, "city", "Tokyo"]
```

Access:

```calcium
person.name   // "Tanaka"
person.age    // 25
```

Separating keys and values:

```calcium
keys(person)    // ["name", "age", "city"]
values(person)  // ["Tanaka", 25, "Tokyo"]
```

Building hashes:

```calcium
k = ["a", "b", "c"]
v = [1, 2, 3]
hash(k, v)  // ["a", 1, "b", 2, "c", 3]
```

## Operators

### Arithmetic Operators

```calcium
+   // Addition (numbers and strings)
-   // Subtraction
*   // Multiplication
/   // Division (always floating point)
%   // Modulo
**  // Exponentiation
```

`+` works with both numbers and strings. For strings, `+` is syntactic sugar for the `concat` function:

```calcium
10 + 11                      // 21
"Hello" + " " + "World"      // "Hello World" (syntactic sugar for concat)
concat("Hello", " ", "World") // "Hello World"
concat("value: ", 42)        // "value: 42"
```

### Comparison Operators

```calcium
==  // Equal
!=  // Not equal
<   // Less than
>   // Greater than
<=  // Less than or equal
>=  // Greater than or equal
```

Chained comparisons are supported:

```calcium
0 <= n <= 150   // Equivalent to 0 <= n && n <= 150
```

### Logical Operators

```calcium
&&  // Logical AND
||  // Logical OR
!   // Logical NOT
```

Logical operators use short-circuit evaluation:
- `a && b`: If `a` is false, returns false without evaluating `b`
- `a || b`: If `a` is true, returns true without evaluating `b`

### Spread Operator

`...` is a postfix operator that expands an array into argument list.

```calcium
func add(x, y) = x + y;

// Spread in pipeline
[2, 3]... |> add           // add(2, 3) -> 5

// Spread in function call
add([2, 3]...)             // add(2, 3) -> 5

// Mixed with regular arguments
func foo(a, b, c) = a + b + c;
foo(1, [2, 3]...)          // foo(1, 2, 3) -> 6

// Works with variables
pair = [10, 20];
pair... |> add             // add(10, 20) -> 30

// Combined with array concatenation
[[1, 2] [3, 4]]... |> sum  // sum(1, 2, 3, 4)
```

### Operator Precedence

From highest to lowest:

| Precedence | Operators | Description |
|------------|-----------|-------------|
| 1 | `f(x)`, `obj.key`, `...` | Function call, member access, spread |
| 2 | `-`, `!` | Unary operators (negation, logical not) |
| 3 | `**` | Exponentiation |
| 4 | `*`, `/`, `%` | Multiplication, division, modulo |
| 5 | `+`, `-` | Addition, subtraction |
| 6 | `<`, `>`, `<=`, `>=` | Comparison |
| 7 | `==`, `!=` | Equality, inequality |
| 8 | `&&` | Logical AND |
| 9 | `||` | Logical OR |
| 10 | `|>`, `!>` | Pipeline |
| 11 | `=` | Assignment |

## Constraints

Constraints define conditions that values must satisfy. They function as an alternative to types.

### Defining Constraints

```calcium
constraint Age(n) = 0 <= n <= 150;
constraint Email(s) = s |> matches(/^.+@.+\..+$/);
constraint Status(s) = s in ["active", "pending", "closed"];
constraint Positive(n) = n > 0;
constraint NonZero(n) = n != 0;
```

`:` is used only to attach constraints to function parameters (described later).

### Constraint Evaluation Timing

Constraints are checked at compile time whenever possible. Values that cannot be determined statically are checked at runtime.

```calcium
func divide(a, b: NonZero?) = a / b;

// Literal values: checked at compile time
divide(10, 0);       // Compile error

// Dynamic values: checked at runtime
x = read_input();
divide(10, x);       // Runtime error if x is 0
```

Behavior when constraint violation occurs at runtime:
- In pure functions (`func`): Program halts (panic)
- In effect functions (`func!`): Returns `failure`

### Checking Constraints

Add `?` to a constraint name to check it. Returns a boolean.

```calcium
25 |> Age?        // true
200 |> Age?       // false
"test@example.com" |> Email?  // true
```

### Combining Constraints

```calcium
constraint PositiveInt(n) = n > 0
constraint Under100(n) = n < 100
constraint Score(n) = n |> PositiveInt? && n |> Under100?
```

### Structural Constraints

```calcium
constraint User(u) =
  u |> has("name") &&
  u |> has("age") &&
  u.age |> Age? &&
  u.name |> len |> (n => n > 0)
```

### Constraint Implementation Details

- Numeric constraints: Validated with comparison operations
- String constraints: Validated with regular expressions
- Enumeration constraints: Validated with containment check

## Functions

### Pure Functions

Functions without side effects are defined with `func`.

```calcium
func add(a, b) = a + b

func double(x) = x * 2

func greet(name) = "Hello, " + name
```

### Effect Functions

Functions with side effects are defined with `func!`.

```calcium
func! save(data) = ...
func! notify(user, message) = ...
func! fetch(url) = ...
```

### Parameter Constraints

```calcium
func divide(a, b: NonZero?) = a / b

func register(name: NonEmpty?, age: Age?) = ...
```

### Required and Rest Parameters

Function parameters can be divided into required and rest parts.

```calcium
func sum(| items) = items |> reduce(+)

func process(first, second | rest) = ...
```

Calling:

```calcium
sum(1, 2, 3, 4, 5)       // items = [1, 2, 3, 4, 5]
process(1, 2, 3, 4, 5)   // first = 1, second = 2, rest = [3, 4, 5]
process(1, 2)            // first = 1, second = 2, rest = []
process(1)               // Error, missing required argument
```

### First-class Functions

Functions can be treated as values.

```calcium
double = (x) => x * 2;
triple = (x) => x * 3;

items |> map(double);
items |> map(triple);
```

### Function Scope and Closures

Functions and lambdas can reference external variables (closures are supported).

```calcium
// Reference parameters
func add(a, b) = a + b;
f = (x) => x * 2;

// Capture external variable (closure)
n = 2;
f = (x) => x * n;   // OK: n is captured
f(5);               // 10

// Return a function that captures parameter
func make_adder(x) = n => x + n;
add5 = make_adder(5);
add5(3);            // 8
```

Closures can be used with higher-order functions:

```calcium
n = 2;
[1, 2, 3] |> map(x => x * n);  // [2, 4, 6]
```

### Shadowing

You can define a variable with the same name in an inner scope (shadowing).

```calcium
x = 10;
func foo(x) = x * 2;  // Parameter x shadows outer x
foo(5);               // 10
```

Since functions can only reference their parameters, shadowing doesn't cause confusion.

### Recursion

Functions can call themselves recursively.

```calcium
func factorial(n) =
  match n
    n <= 0 => 1
    _ => n * factorial(n - 1);

func sum_list(xs) =
  match len(xs)
    0 => 0
    _ => xs[0] + sum_list(array.slice(xs, 1));
```

Tail recursion optimization is at the implementation's discretion (not mandated by the specification).

## Branching

### match

Performs pattern branching based on values.

```calcium
func describe(x) =
  match x
    0 => "zero"
    n: Positive? => "positive"
    n: Negative? => "negative";
```

Conditions can be written directly:

```calcium
func process(x) =
  match x
    0 < n < 100 => n * 2
    n > 0 => 100
    _ => 0;
```

Wildcard `_` matches everything else:

```calcium
func to_string(x) =
  match x
    0 => "zero"
    1 => "one"
    _ => "other";
```

```calcium
func handle(result) =
  match result
    success(v) => v |> process
    failure(e) => e |> log
```

### Exhaustiveness Check

Match expressions are checked for exhaustiveness:

- Having wildcard `_` makes it exhaustive
- Covering both `success`/`failure` makes it exhaustive
- Otherwise, wildcard is required (compiler warning)

```calcium
// OK: Has wildcard
match x
  0 => "zero"
  _ => "other"

// OK: Covers both success/failure
match result
  success(v) => v
  failure(e) => default

// Warning: May not be exhaustive
match x
  0 => "zero"
  1 => "one"
  // Warning because _ is missing
```

## Iteration

### map

Applies a function to each element of an array.

```calcium
numbers = [1, 2, 3, 4, 5]
doubled = numbers |> map(x => x * 2)  // [2, 4, 6, 8, 10]
```

### filter

Keeps only elements that satisfy a condition.

```calcium
numbers = [1, 2, 3, 4, 5]
evens = numbers |> filter(x => x % 2 == 0)  // [2, 4]
```

### reduce

Aggregates an array into a single value.

```calcium
numbers = [1, 2, 3, 4, 5];
total = numbers |> reduce((a, b) => a + b);  // 15
```

Initial value can be specified:

```calcium
[1, 2, 3] |> reduce((a, b) => a + b);        // 6 (first element is initial)
[1, 2, 3] |> reduce((a, b) => a + b, 0);     // 6 (initial specified)

[] |> reduce((a, b) => a + b);               // Error (empty array + no initial)
[] |> reduce((a, b) => a + b, 0);            // 0 (returns initial)
```

### Asynchronous Processing: core.async!

The `core.async!` module provides primitives for asynchronous processing.
All async-related functionality is provided as functions of this module.

#### async.stay

`async.stay` is a function for event waiting with state. Can only be used inside `func!`.
This is the only place in Calcium where state can be handled.

```calcium
use core.async!
use core.io!
use core.schedule!

func! main() =
  task1 = async.spawn(() => fetch(url1))
  task2 = async.spawn(() => fetch(url2))

  async.stay(results: []) {
    task1.done
      |> async.expects((r) => {
        async.continue(results: [results [r]])
      })
      |> _.ready()

    task2.done
      |> async.expects((r) => {
        async.continue(results: [results [r]])
      })
      |> _.ready()

    schedule.timeout(5000)
      |> async.expects(() => {
        async.leave("timeout")
      })
      |> _.ready()

    async.all([task1, task2]).done
      |> async.expects(() => {
        async.leave(results)
      })
      |> _.ready()
  } !? {
    success(r) => log(r)
    failure(e) => log_error(e)
  };
```

#### async.spawn

`async.spawn` is an effect function that starts a background task. Can only be used inside `func!`.

```calcium
task = async.spawn(() => fetch(url))   // spawn func!
task = async.spawn(() => compute(data)) // func can also be spawned
```

- `async.spawn` itself is an effect (only in `func!`)
- The argument function can be either pure (`func`) or impure (`func!`)
- Return value is of type `Task<a>`

#### async.expects and Handler<a>

`async.expects` defines an event handler and returns a value of type `Handler<a>`.
Use with pipeline operator combined with an event source.

```calcium
// event_source |> async.expects(handler_function) |> _.ready()
task.done
  |> async.expects((result) => {
    async.continue(results: [results [result]])
  })
  |> _.ready()

// Bind to variable and activate later
handler = task.done
  |> async.expects((result) => {
    async.continue(results: [results [result]])
  })
handler.ready()
```

Handler has the following states:

```
dormant ──ready()──> active ──async.cancel()──> cancelled
                        ^                           |
                        └─────────ready()───────────┘

active ──pause()──> paused ──resume()──> active
       <─reset()─┘
```

Handler methods:

| Method | Description |
|--------|-------------|
| `.ready()` | Activate handler. Bound to current `stay`. Returns self (chainable). No-op if already active |
| `.reset()` | Cancel + reactivate. Used for timer reset etc. |
| `.pause()` | Pause (ignore events) |
| `.resume()` | Resume from pause |

Handler can be defined outside `stay` (dormant state). When `.ready()` is called, it binds to the current `stay`, and `async.leave`/`async.continue` act on that `stay`.

```calcium
// Create handler with factory function
make_timeout = (ms, msg) =>
  schedule.timeout(ms)
    |> async.expects(() => { async.leave(msg) })

func! main() =
  async.stay(count: 0) {
    timeout = make_timeout(5000, "timeout");
    timeout.ready();

    io.stdin
      |> async.expects((line) => {
        match line
          "cancel" => async.cancel(timeout); async.continue(count: count)
          _ => async.continue(count: count + 1)
      })
      |> _.ready()
  }
```

#### async.continue

Updates state and returns to waiting.

```calcium
async.continue(count: count + 1)
async.continue(results: [results [r]], count: count + 1)
```

#### async.leave

Exits the `stay` loop. By default, all Tasks spawned within stay are auto-cancelled.

```calcium
async.leave(value)                      // Auto-cancel all Tasks
async.leave(value, keeping: [task1])    // Only specified Tasks continue

// To keep all Tasks, manage them yourself
tasks = [task1, task2, task3]
async.leave(value, keeping: tasks)
```

| Call | Behavior |
|------|----------|
| `async.leave(value)` | Auto-cancel all Tasks in stay |
| `async.leave(value, keeping: [tasks])` | Only specified Tasks continue, rest cancelled |

#### async.all

Wait for completion of multiple tasks.

```calcium
async.all([task1, task2]).done
  |> async.expects(() => {
    async.leave(results)
  })
  |> _.ready()
```

#### async.cancel

Cancel a Task or Handler.

```calcium
async.cancel(handler)  // Only release Handler, associated Task continues
async.cancel(task)     // Interrupt Task + release all Handlers below
```

Cancel hierarchy:

```
Task (created by async.spawn)
  └── Handler (created by async.expects)
        └── Handler
        └── Handler
```

- `async.cancel(handler)`: Only release Handler. Task continues
- `async.cancel(task)`: Interrupt Task, all Handlers waiting for that Task are also released

```calcium
task1 = async.spawn(() => fetch(url1))

async.stay(results: []) {
  h1 = task1.done
    |> async.expects((r) => { async.continue(results: [results [r]]) })
  h1.ready()

  h2 = task1.done
    |> async.expects((r) => { log(r); async.continue(results: results) })
  h2.ready()

  schedule.timeout(100)
    |> async.expects(() => {
      async.cancel(h1)  // Only release h1, task1 and h2 continue
      async.continue(results: results)
    })
    |> _.ready()

  schedule.timeout(500)
    |> async.expects(() => {
      async.cancel(task1)  // Interrupt task1 + release both h1, h2
      async.leave("cancelled")
    })
    |> _.ready()
}
```

#### core.async! Function List

| Function | Description |
|----------|-------------|
| `async.stay(state) { ... }` | Start event loop |
| `async.spawn(fn)` | Start background task -> `Task<a>` |
| `async.expects(fn)` | Create event handler -> `Handler<a>` |
| `async.continue(state)` | Update state and return to waiting |
| `async.leave(value)` | Exit loop (cancel all Tasks) |
| `async.leave(value, keeping: [...])` | Only specified Tasks continue |
| `async.cancel(target)` | Cancel Task/Handler |
| `async.all(tasks)` | Combine multiple tasks |

#### Concurrency Model

- Single-threaded event loop
- Multiple Handlers wait concurrently but processing is sequential
- When multiple fire simultaneously, the first matched one executes
- Next event is not processed until current event processing completes

### I/O: core.io!

The `core.io!` module provides input/output functionality.

#### io.println / io.say / io.print

Output to standard output.

| Function | Description |
|----------|-------------|
| `io.println` | Output with newline |
| `io.say` | Synonym for `io.println` |
| `io.print` | Output without newline |

```calcium
use core.io!

"Hello" !> io.println;  // Hello\n (with newline)
"Hello" !> io.say;      // Hello\n (synonym for io.println)
"Hello" !> io.print;    // Hello (no newline)

// Values are automatically converted to strings
42 !> io.println;       // 42\n
[1, 2, 3] !> io.say;    // [1, 2, 3]\n
```

#### io.stdin / io.eof (Event Sources)

```calcium
use core.io!
use core.async!

async.stay(lines: []) {
  io.stdin
    |> async.expects((line) => {
      async.continue(lines: [lines [line]])
    })
    |> _.ready()

  io.eof
    |> async.expects(() => {
      async.leave(lines)
    })
    |> _.ready()
}
```

#### io.stdin

Event source that reads one line from standard input.

```calcium
io.stdin
  |> async.expects((line) => {
    // line is bound to the read line
    process(line);
    async.continue(state: state)
  })
  |> _.ready()
```

#### io.eof

Event source that detects end of standard input (EOF).

```calcium
io.eof
  |> async.expects(() => {
    async.leave(result)
  })
  |> _.ready()
```

### Scheduling: core.schedule!

The `core.schedule!` module provides time-based events.

```calcium
use core.schedule!
use core.async!

async.stay(count: 0) {
  schedule.timeout(5000)
    |> async.expects(() => {
      async.leave("timeout")
    })
    |> _.ready()

  schedule.interval(1000)
    |> async.expects(() => {
      log(count);
      async.continue(count: count + 1)
    })
    |> _.ready()
}
```

#### schedule.timeout

Event source that fires once after specified milliseconds.

```calcium
schedule.timeout(5000)
  |> async.expects(() => {
    async.leave("timeout")
  })
  |> _.ready()
```

Timer can be reset with `handler.reset()`:

```calcium
timeout = schedule.timeout(5000)
  |> async.expects(() => { async.leave("timeout") })
timeout.ready()

io.stdin
  |> async.expects((line) => {
    timeout.reset()  // Reset timer (fires again after 5 seconds)
    async.continue(state: state)
  })
  |> _.ready()
```

#### schedule.interval

Event source that fires repeatedly every specified milliseconds.

```calcium
schedule.interval(1000)
  |> async.expects(() => {
    log("tick");
    async.continue(state: state)
  })
  |> _.ready()
```

## Pipelines

### Pure Pipeline `|>`

Flows values from left to right, applying functions.

```calcium
result = data
  |> transform
  |> validate
  |> format
```

### Effect Pipeline `!>`

Chains operations with side effects. Can only be used inside `func!`.

```calcium
func! save_and_notify(data) =
  data
  !> save
  !> notify
  !? {
    success(_) => done()
    failure(e) => log_error(e)
  }
```

### Error Handling `!?`

Must handle errors at the end of an effect pipeline.

```calcium
data
!> save
!> notify
!? {
  success(result) => result |> log
  failure(err) => err |> log_error
}
```

When using `!>`, must always terminate with `!?`.

## Namespaces

### Definition

```calcium
// math.ca
namespace math;

func add(a, b) = a + b;
func multiply(a, b) = a * b;
constraint Positive(n) = n > 0;
```

### Usage

```calcium
// main.ca
use math;

result = 5 |> math.add(3) |> math.multiply(2);
```

Import only specific elements:

```calcium
use math { add, Positive };

result = 5 |> add(3);
x: Positive? = 10;
```

Nested namespaces:

```calcium
namespace util.string;

func trim(s) = ...;
func upper(s) = ...;
```

```calcium
use util.string { trim };

" hello " |> trim;
```

### Module Resolution

All `use` statements are resolved relative to the entry point directory (project root).

```
project/
  common/
    utils.ca
  features/
    auth/
      login.ca    <- use common.utils; works
  main.ca         <- Entry point
```

```calcium
// features/auth/login.ca
use common.utils;   // References project/common/utils.ca
```

Resolution order:
1. Relative path from project root (entry point directory)
2. Standard library

## Error Handling

### Compile-time Errors

Constraint violations are detected at compile time.

```calcium
func divide(a, b: NonZero?) = a / b

divide(10, 0)  // Compile error: 0 does not satisfy NonZero?
```

### Runtime Errors (Side Effects)

Errors from side effects are handled with `!?`.

```calcium
func! main() =
  data |> validate
  !> save_to_db
  !> notify_user
  !? {
    success(result) => result |> log
    failure(err) => err |> log_error
  }
```

## Complete Example

```calcium
// user.ca
namespace user

constraint Age(n) = 0 <= n <= 150
constraint Email(s) = s |> matches(/^.+@.+\..+$/)
constraint NonEmpty(s) = s |> len |> (n => n > 0)

constraint User(u) =
  u |> has("name") &&
  u |> has("age") &&
  u |> has("email") &&
  u.name |> NonEmpty? &&
  u.age |> Age? &&
  u.email |> Email?

func create(name, age, email) =
  ["name", name, "age", age, "email", email]

func validate(u) =
  match u |> User?
    false => failure("invalid user")
    true => success(u)
```

```calcium
// main.ca
use user
use core.io!

func! main() =
  input = ["name", "Tanaka", "age", 25, "email", "tanaka@example.com"]

  result = input
    |> user.validate
    |> match
        success(u) => u
        failure(e) => leave e

  result
  !> save
  !> notify
  !? {
    success(_) => "Done!" !> io.say
    failure(e) => concat("Error: ", e) !> io.say
  }
```

## Standard Functions

### Global Functions (no use required)

```calcium
len(xs)                  // Length of array or string
get(collection, key)     // Safe access -> success/failure
get(collection, key, default)  // With default value
has(collection, key)     // Whether key or index exists -> true/false
                         // Hash: has(person, "name")
                         // Array: has(numbers, 5)  // Whether index 5 exists
concat(s1, s2, ...)      // String concatenation
to_string(x)             // Convert any value to string
to_num(s)                // Convert string to number -> success/failure
matches(s, regex)        // Regex match -> true/false
keys(hash)               // Array of hash keys
values(hash)             // Array of hash values
hash(keys, values)       // Create hash from key array and value array
```

### string Namespace

```calcium
use string;

string.trim(s)           // Remove leading/trailing whitespace
string.upper(s)          // Convert to uppercase
string.lower(s)          // Convert to lowercase
string.split(s, sep)     // Split -> array
string.join(xs, sep)     // Join -> string
```

### math Namespace

```calcium
use math;

math.floor(n)            // Floor
math.ceil(n)             // Ceiling
math.round(n)            // Round
math.abs(n)              // Absolute value
math.max(a, b)           // Maximum
math.min(a, b)           // Minimum
```

### array Namespace

See the array operations section.

## Standard I/O

### Output

Output is provided by the `core.io!` module:

```calcium
use core.io!

"Hello" !> io.println;  // Hello\n (with newline)
"Hello" !> io.say;      // Hello\n (synonym for io.println)
"Hello" !> io.print;    // Hello (no newline)
```

### Input

Input is obtained via the `io.stdin` event in the `core.io!` module:

```calcium
use core.async!
use core.io!
use core.schedule!

async.stay(lines: []) {
  io.stdin
    |> async.expects((line) => {
      async.continue(lines: [lines [line]])
    })
    |> _.ready()

  schedule.timeout(1000)
    |> async.expects(() => {
      async.leave(lines)
    })
    |> _.ready()
} !? {
  success(result) => result |> map(line => line !> io.say)
  failure(e) => e !> io.say
};
```

See the "I/O: core.io!" section for details.

## Entry Point

Top-level code is executed directly. No `main` function is required.

```calcium
use core.io!

// File contents are executed directly
x = 10;
y = 20;
x + y !> io.say;
```

Command-line arguments are obtained via the global variable `args` (program name not included):

```calcium
use core.io!

args[0];              // First argument
len(args);            // Number of arguments

// Output all arguments
args |> map(arg => arg !> io.say);
```

Program name is obtained via `program_name`:

```calcium
use core.io!

program_name !> io.say;  // Name of the running program
```

## File Extension

`.ca` or `.calcium`

## Reserved Words

```
func func! constraint namespace use
match
map filter reduce
in has keys values hash len
success failure
return
```

Note: Async-related features (`stay`, `spawn`, `expects`, `continue`, `leave`, `cancel`, etc.) are not language keywords but functions provided by the `core.async!` module. Similarly, `timeout` and `interval` are provided by `core.schedule!`, and `stdin` is provided by the `core.io!` module.
