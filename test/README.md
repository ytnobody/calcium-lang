# calcium-test

TAP-compatible test framework for the [Calcium](https://github.com/ytnobody/calcium-lang) programming language, inspired by Perl's [Test::More](https://metacpan.org/pod/Test::More).

## Installation

```
bone add ytnobody/test
```

## Usage

```calcium
use test;

ctx = test.new();
ctx = test.plan(ctx, 3);
ctx = test.is(ctx, "addition", 1 + 1, 2);
ctx = test.ok(ctx, "truthy check", true);
ctx = test.isnt(ctx, "not equal", 1, 2);
test.done(ctx);
```

Output:

```
1..3
ok 1 - addition
ok 2 - truthy check
ok 3 - not equal
# Tests: 3
# Passed: 3
# Failed: 0
```

## API

### Context Management

#### `new()`

Creates a fresh test context. All test functions take a context as the first argument and return an updated context.

```calcium
ctx = test.new();
// => {total: 0, passed: 0, failed: 0, planned: 0}
```

#### `plan(ctx, n)`

Declares the expected number of tests. Prints `1..N` in TAP format. If the actual test count doesn't match at `done()`, a diagnostic message is printed.

```calcium
ctx = test.plan(ctx, 5);
// Output: 1..5
```

### Assertions

All assertion functions print a TAP result line immediately and return the updated context.

#### `ok(ctx, label, value)`

Asserts that `value` is truthy (not `false`, `null`, `0`, or `""`).

```calcium
ctx = test.ok(ctx, "truthy check", true);
// Output: ok 1 - truthy check
```

#### `is(ctx, label, actual, expected)`

Asserts that `actual` equals `expected`.

```calcium
ctx = test.is(ctx, "addition", 1 + 1, 2);
// Output: ok 1 - addition
```

On failure, diagnostic output shows the actual and expected values:

```
not ok 1 - addition
#   got:      3
#   expected: 2
```

#### `isnt(ctx, label, actual, expected)`

Asserts that `actual` does not equal `expected`.

```calcium
ctx = test.isnt(ctx, "not equal", 1, 2);
// Output: ok 1 - not equal
```

#### `pass(ctx, label)`

Unconditionally records a passing test.

```calcium
ctx = test.pass(ctx, "always passes");
// Output: ok 1 - always passes
```

#### `fail(ctx, label)`

Unconditionally records a failing test.

```calcium
ctx = test.fail(ctx, "always fails");
// Output: not ok 1 - always fails
#   explicitly failed
```

### Summary

#### `done(ctx)`

Prints the test summary and checks the plan. Call this at the end of your test file.

```calcium
test.done(ctx);
// Output:
// # Tests: 5
// # Passed: 4
// # Failed: 1
// # Looks like you planned 5 tests but ran 4
```

## Design Notes

Calcium is a functional language with no mutable state. The test module uses a **context threading** pattern: each assertion function takes a context hash as its first argument and returns an updated context. This allows tracking test counts across assertions without mutation.

Each assertion is a side-effectful function (`func!`) that prints TAP output immediately while returning the updated context for the next assertion.

## TAP Compatibility

Output follows the [Test Anything Protocol](https://testanything.org/):

- Plan line: `1..N`
- Pass: `ok N - label`
- Fail: `not ok N - label`
- Diagnostics: `# ...`

## License

MIT
