# Calcium

A functional programming language with pipelines, pattern matching, and closures.

## Features

- **Pipeline Operator** (`|>`) - Chain function calls in a readable left-to-right style
- **Pattern Matching** - Powerful `match` expressions for control flow
- **First-class Functions** - Lambda expressions and closures with proper variable capture
- **Partial Application** - Built-in support for curried functions like `map`, `filter`, `reduce`
- **Array Destructuring** - Unpack arrays into variables with `[a, b, c] = arr` or `[head | tail] = arr`
- **Hashes** - Associative arrays with dot and bracket access
- **Constraints** - Define and enforce value constraints with `constraint` and `?` operator
- **Chained Comparisons** - Write `0 <= x <= 100` instead of `x >= 0 && x <= 100`
- **Effect Functions** - Distinguish pure functions from side-effecting ones with `func!`
- **Module System** - Organize code with `use` statements

## Installation

```bash
go build -o calcium ./cmd/calcium
```

## Usage

```bash
# Run a Calcium program
calcium hello.ca
calcium run examples/pipeline.ca

# Start interactive REPL
calcium repl

# Show version
calcium version
```

## Quick Start

### Hello World

```calcium
use core.io;

"Hello, Calcium!" |> io.say;
```

### Pipeline Operator

The pipeline operator `|>` passes the left-hand value as the first argument to the right-hand function:

```calcium
use core.io;

// These are equivalent:
io.say("Hello");
"Hello" |> io.say;

// Chain multiple operations
[1, 2, 3, 4, 5]
    |> filter(x => x % 2 == 1)
    |> map(x => x * x)
    |> reduce((a, b) => a + b, 0)
    |> io.say;  // Output: 35
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
    true => n * factorial(n - 1);
```

### Pattern Matching

```calcium
func fizzbuzz(n) = match true
    n % 15 == 0 => "FizzBuzz"
    n % 3 == 0 => "Fizz"
    n % 5 == 0 => "Buzz"
    true => to_string(n);

func fib(n) = match n
    0 => 0
    1 => 1
    true => fib(n - 1) + fib(n - 2);
```

### Partial Application

`map`, `filter`, and `reduce` support partial application, making them ideal for pipelines:

```calcium
// Full application
map(x => x * 2, [1, 2, 3]);  // [2, 4, 6]

// Partial application returns a function
double_all = map(x => x * 2);
double_all([1, 2, 3]);  // [2, 4, 6]

// Perfect for pipelines
[1, 2, 3, 4, 5]
    |> filter(x => x > 2)
    |> map(x => x * 10)
    |> io.say;  // [30, 40, 50]
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
use core.array;

numbers = [1, 2, 3, 4, 5];

// Built-in functions
map(x => x * 2, numbers);           // [2, 4, 6, 8, 10]
filter(x => x > 2, numbers);        // [3, 4, 5]
reduce((a, b) => a + b, 0, numbers); // 15

// Array module functions
numbers |> array.reverse;  // [5, 4, 3, 2, 1]
numbers |> array.sum;      // 15
array.take(numbers, 3);    // [1, 2, 3]

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
    true => head(arr) + sum_list(tail(arr));

// Or with destructuring in loops
[x | xs] = numbers;
// Process x, then continue with xs
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

// Built-in hash functions
keys(person);      // ["name", "age", "city"]
values(person);    // ["Alice", 30, "Tokyo"]
len(person);       // 3
```

### Strings

```calcium
use core.string;

message = "  Hello, World!  ";

message |> string.trim;                    // "Hello, World!"
message |> string.upper;                   // "  HELLO, WORLD!  "
message |> string.lower;                   // "  hello, world!  "
string.split("a,b,c", ",");                // ["a", "b", "c"]
string.join(["a", "b", "c"], "-");         // "a-b-c"
string.contains(message, "World");         // true
string.replace(message, "World", "Calcium"); // "  Hello, Calcium!  "
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

// Handle results with !?
42 |> Positive? !? {
    success(v) => "Valid!"
    failure(v) => "Invalid!"
};

// Use constraints in function parameters
func! safe_divide(x, y: Positive?) = x / y;

safe_divide(10, 2);   // success(5)
safe_divide(10, 0);   // failure(0)
safe_divide(10, -1);  // failure(-1)
```

## Standard Library

### core.io

| Function | Description |
|----------|-------------|
| `io.say(value)` | Print value with newline |
| `io.print(value)` | Print value without newline |

### core.array

| Function | Description |
|----------|-------------|
| `array.reverse(arr)` | Reverse an array |
| `array.sum(arr)` | Sum all elements |
| `array.take(arr, n)` | Take first n elements |
| `array.drop(arr, n)` | Drop first n elements |

### core.string

| Function | Description |
|----------|-------------|
| `string.trim(s)` | Remove leading/trailing whitespace |
| `string.upper(s)` | Convert to uppercase |
| `string.lower(s)` | Convert to lowercase |
| `string.split(s, sep)` | Split string by separator |
| `string.join(arr, sep)` | Join array with separator |
| `string.contains(s, sub)` | Check if string contains substring |
| `string.starts_with(s, prefix)` | Check if string starts with prefix |
| `string.ends_with(s, suffix)` | Check if string ends with suffix |
| `string.replace(s, old, new)` | Replace occurrences |

### Built-in Functions

| Function | Description |
|----------|-------------|
| `map(fn, arr)` | Apply function to each element |
| `filter(pred, arr)` | Keep elements matching predicate |
| `reduce(fn, init, arr)` | Fold array to single value |
| `range(start, end)` | Generate array of integers |
| `len(arr)` | Get array/hash length |
| `concat(...)` | Concatenate strings |
| `to_string(value)` | Convert to string |
| `keys(hash)` | Get all keys from hash |
| `values(hash)` | Get all values from hash |

## Examples

See the `examples/` directory:

- `hello.ca` - Hello World
- `factorial.ca` - Factorial with pattern matching
- `fibonacci.ca` - Fibonacci sequence
- `pipeline.ca` - Pipeline operator demonstrations
- `strings.ca` - String manipulation
- `hash.ca` - Hash (associative array) operations
- `constraint.ca` - Constraint validation examples

Run an example:

```bash
calcium examples/pipeline.ca
```

## License

MIT
