# Native Boundary (FFI) Specification

Status: **normative** for the primitive layer. This document defines the exact
surface that cannot be written in Calcium and must therefore be supplied by the
host runtime. It is the contract a self-hosted implementation has to satisfy.

See `selfhost-scope.md` for which components are in scope for self-hosting and
`bone-format.md` for the serialised program format.

---

## 1. Three native layers

There are three distinct native surfaces. Conflating them is the most common
source of confusion when reimplementing the runtime.

| Layer | What it is | Self-hostable? | Count |
|---|---|---|---|
| **A. VM builtins** | Names the *compiler* treats as builtins: `len`, `concat`, `map`, ... | No — part of the VM core | 16 |
| **B. `__` primitives** | Low-level functions wrapping OS/library facilities. Called by the Calcium standard library. | No — permanent native boundary | 78 |
| **C. Module intrinsics** | Values the VM injects into modules: event sources (`stdin`, `eof`), async operations, and the Go-side `core.io` exports. | No — part of the VM core | ~30 |

Everything else — `pkg/vm/stdlib/core/*.ca`, 893 lines — is Calcium source and
is *not* part of this contract.

### 1.1 Layer A: compiler-known VM builtins

```
len  concat  to_string  get  has  head  tail  push  range  map  filter
reduce  keys  values  hash_set  hash_merge
```

These are recognised by name in the compiler (`pkg/compiler/compiler.go`) and
dispatched by the VM. `map`, `filter` and `reduce` need the VM to call back into
user closures, so they cannot be moved to Calcium without a performance cliff.

### 1.2 Layer C: module intrinsics

Injected by `initModules` in `pkg/vm/vm.go`:

| Module | Intrinsics |
|---|---|
| `core.io` | `print`, `println`, `say`, `read_file`, `write_file`, `format`, `list_dir`, `mkdir`, `exists`, `delete_file`, `file_info` |
| `core.io` (event sources) | `stdin`, `eof` |
| `core.async` | `spawn`, `expects`, `continue`, `leave`, `cancel`, `all`, `channel`, `timeout`, `interval` |

Note that `core.io` exists twice: once as Go exports in `initModules` and once
as `stdlib/core/io.ca` delegating to `__` primitives. **The Go exports win**: a
module already registered in the in-memory module map is never loaded from
`.ca`, so `stdlib/core/io.ca` is unreachable. It shows up as missing exports
(`io.read_lines` does not exist) rather than as a shadowing ambiguity.

Consequence for a reimplementation: for `core.io`, provide the exports listed
above rather than compiling the `.ca` file. Other `core.*` modules
(`core.string`, `core.array`, `core.math`, `core.time`, `core.bytes`, ...) are
not pre-registered and do load from `.ca`.

---

## 2. Calling convention

Primitives are ordinary functions whose names are prefixed with `__`:

* **Arity is checked by the primitive itself**, not the compiler. Each returns
  a `failure(<message>)` for a wrong argument count or wrong argument type. The
  message always starts with the primitive name, e.g.
  `"__str_split: arguments must be strings"`.
* **Arity is fixed or a small range.** Ranges exist so that optional trailing
  arguments can be omitted (`__http_post(url, body[, content_type][, headers])`).
* **No primitive throws.** Every recoverable error is a `failure` value. Only a
  bug in the runtime should abort execution.
* **Error mode is inconsistent by design, and must be reproduced exactly:**

  | Mode | Meaning | Examples |
  |---|---|---|
  | `failure(msg)` + raw success value | Most total functions and most I/O | `__str_split` → `Array`; `__exists` → `Bool`; `__http_get` → `Hash`; `__time_components` → `Hash` |
  | `failure(msg)` + `success(v)` | Operations whose result is explicitly fallible | `__read_file` → `success(String)`; `__to_int` → `success(Int)`; `__json_parse` → `success(value)`; `__proc_start` → `success(Int)` |
  | `failure(msg)` + `Maybe` | Lookups that can legitimately miss | `__str_char_at` → `Maybe.Some(String)` / `Maybe.None` |
  | plain value, no error channel | Total, infallible functions | `__str_len` → `Int`; `__str_trim` → `String`; `__type_of` → `String`; `__sin` → `Float` |

  A reimplementation must match the mode, not just the value: user code pattern
  matches on `success(...)` / `failure(...)` / `Maybe`.
* `__print` and `__println` return `Maybe.None`.

### 2.1 Obligations the runtime must honour

These are behavioural, not just signatures:

1. **Strings are sequences of Unicode code points.** `__str_len` counts runes,
   `__str_substring` and `__str_char_at` index by code point, not byte offset.
2. **Integer/float distinction is preserved.** `__to_int` truncates; numeric
   primitives that accept "number" accept both int and float.
3. **Timeout/interval values are milliseconds.** `__time_*` timestamps are Unix
   seconds, except `__time_now_ms` which is Unix milliseconds.
4. **`__os_exit` terminates after reaping child processes.** Tracked
   subprocesses must not outlive the parent on any exit path.
5. **Secret-carrying primitives** (`__http_post_secret`, `__proc_write_secret`)
   are the only ones allowed to unwrap a `Secret`. They verify the supplied
   capability's sink and scope first and return `failure` when it does not
   authorise the target. An unprivileged primitive must never accept a `Secret`
   argument.
6. **`__proc_*` handles are opaque integers.** A handle from a different VM or a
   closed process is `failure("invalid handle")`.

---

## 3. Primitive catalogue

`R` = raw value returned on success. `S(v)` = `success(v)`. `M` = `Maybe`.
All failures are `failure(String)`.

### 3.1 Output and filesystem

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__print` | 1 | any (printed via `to_string`) | `Maybe.None` |
| `__println` | 1 | any | `Maybe.None` |
| `__read_file` | 1 | path: String | `S(String)` |
| `__write_file` | 2 | path: String, content: String | `S(Maybe.None)` |
| `__list_dir` | 1 | path: String | `S(Array<String>)` |
| `__mkdir` | 1 | path: String | `S(Maybe.None)` |
| `__exists` | 1 | path: String | `R(Bool)`; `failure` on permission error |
| `__delete_file` | 1 | path: String | `S(Maybe.None)` |
| `__file_info` | 1 | path: String | `S(Hash)` — keys `name`, `size`, `is_dir`, `modified` (RFC3339 UTC) |

`__list_dir` returns entries sorted by name (`os.ReadDir` ordering, byte-wise).
`__mkdir` creates parents.

### 3.2 Math

All take numbers. `__floor`, `__ceil` and `__round` return `R(Int)`; the rest
return `R(Float)`.

| Primitive | Arity | Arguments | Notes |
|---|---|---|---|
| `__floor` `__ceil` `__round` | 1 | number | `R(Int)` |
| `__sqrt` | 1 | number | `R(Float)`; `failure` when the argument is negative |
| `__pow` | 2 | base: number, exp: number | `R(Float)` |
| `__sin` `__cos` `__tan` | 1 | number (radians) | `R(Float)` |
| `__log` | 1 | number | `R(Float)`; `failure` unless the argument is positive |
| `__exp` | 1 | number | `R(Float)` |

### 3.3 Strings

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__str_len` | 1 | String | `R(Int)` (code points) |
| `__str_split` | 2 | String, String (sep) | `R(Array<String>)` |
| `__str_join` | 2 | Array, String (sep) | `R(String)` |
| `__str_trim` | 1 | String | `R(String)` (Unicode whitespace) |
| `__str_upper` `__str_lower` | 1 | String | `R(String)` |
| `__str_replace` | 3 | String, old, new | `R(String)` (all occurrences) |
| `__str_substring` | 2–3 | String, start: Int[, end: Int] | `R(String)` — `end` exclusive; omitted = to end |
| `__str_char_at` | 2 | String, index: Int | `M` — `Maybe.Some(String)` / `Maybe.None` |
| `__str_from_code` | 1 | Int (code point) | `S(String)`; fails on surrogate or out-of-range |
| `__str_index_of` | 2 | String, substring | `R(Int)` — `-1` when absent |
| `__str_starts_with` | 2 | String, prefix | `R(Bool)` |
| `__str_ends_with` | 2 | String, suffix | `R(Bool)` |
| `__str_contains` | 2 | String, substring | `R(Bool)` |
| `__str_to_bytes` | 1 | String | `R(Array<Int>)` — UTF-8 bytes, each 0-255 |
| `__str_from_bytes` | 1 | Array of Int (0-255) | `R(String)` — bytes copied verbatim; `failure` for a non-int element or a byte outside 0-255 |
| `__int_to_bytes` | 1 | Int | `R(Array<Int>)` — eight big-endian two's-complement bytes; `failure` for a non-int |
| `__float_to_bytes` | 1 | Float | `R(Array<Int>)` — eight big-endian IEEE 754 bytes; `failure` for a non-float |

### 3.4 Regex

The `Regex` argument is a `Regex` value (constant tag 6), not a string.

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__regex_test` | 2 | String, Regex | `R(Bool)` |
| `__regex_find` | 2 | String, Regex | `S(String)`; `failure("no match")` when absent |
| `__regex_find_all` | 2 | String, Regex | `R(Array<String>)` (empty when none) |
| `__regex_replace` | 3 | String, Regex, replacement | `R(String)` (all matches) |
| `__regex_replace_first` | 3 | String, Regex, replacement | `R(String)` |
| `__regex_split` | 2 | String, Regex | `R(Array<String>)` |
| `__regex_capture` | 2 | String, Regex | `S(Array<String>)` of capture groups |
| `__regex_compile` | 1 | String (pattern, Calcium syntax) | `S(Regex)` / `failure` |

### 3.5 Conversion and reflection

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__to_int` | 1 | Int / Float / String / Bool | `S(Int)`; fails on unparsable string |
| `__to_float` | 1 | Int / Float / String | `S(Float)`; fails on unparsable string |
| `__type_of` | 1 | any | `R(String)` — the Calcium type name |

### 3.6 JSON / TOML

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__json_parse` | 1 | String | `S(value)`; `failure` on syntax error or trailing content |
| `__json_stringify` | 1 | any | `S(String)`; `failure` for unsupported shapes |
| `__toml_parse` | 1 | String | `S(value)` |
| `__toml_stringify` | 1 | Hash | `S(String)` / `failure` |

JSON `null` maps to `Maybe.None` and `Maybe.None` stringifies to `null`.
`Maybe.Some(v)` stringifies as `v`.

### 3.7 HTTP (effectful)

Responses are a `Hash` with keys `status` (Int), `status_text` (String),
`headers` (Hash), `body` (String), `ok` (Bool). Returned **directly**, not
wrapped in `success`; transport errors are `failure`.

| Primitive | Arity | Arguments |
|---|---|---|
| `__http_get` | 1–2 | url: String[, headers: Hash] |
| `__http_delete` | 1–2 | url: String[, headers: Hash] |
| `__http_post` `__http_put` | 2–4 | url, body[, content_type][, headers] |
| `__http_request` | 1 | options Hash: `method`, `url`, optional `headers`, `body` |
| `__http_post_secret` | 5 | url, secret: Secret, capability, content_type, headers |

`__http_post_secret` is the only HTTP path that unwraps a `Secret` and requires
a capability authorising sink `"http"` with `scope.url == url`.

### 3.8 Time

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__time_now` | 0 | — | `R(Int)` Unix seconds |
| `__time_now_ms` | 0 | — | `R(Int)` Unix milliseconds |
| `__time_format` | 2–3 | timestamp: Int, format: String[, tz: String] | `S(String)` |
| `__time_parse` | 2–3 | value: String, format: String[, tz] | `S(Int)` |
| `__time_components` | 1–2 | timestamp: Int[, tz] | `R(Hash)` — `year`, `month`, `day`, `hour`, `minute`, `second`, `weekday`, `yearday` |
| `__time_from_components` | 6–7 | year, month, day, hour, minute, second[, tz] | `R(Int)` |
| `__time_add` | 2 | timestamp: Int, seconds: Int | `R(Int)` |
| `__time_diff` | 2 | timestamp1, timestamp2 | `R(Int)` — `t1 - t2` |

Format strings use the Go reference layout (`2006-01-02 15:04:05`), not
`strftime`. This is a deliberate format-compatibility decision: a replacement
runtime must implement the same layout tokens.

### 3.9 Environment and process control

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__env_get` | 1 | name: String | `S(String)`; `failure("not set")` when absent |
| `__env_set` | 2 | name, value | `S(Maybe.None)` |
| `__env_unset` | 1 | name | `S(Maybe.None)` |
| `__env_list` | 0 | — | `R(Hash)` of all variables |
| `__os_args` | 0 | — | `R(Array<String>)` |
| `__os_exit` | 1 | code: Int | does not return |

### 3.10 External processes

| Primitive | Arity | Arguments | Success result |
|---|---|---|---|
| `__proc_start` | 2 | command: String, args: Array<String> | `S(Int)` handle |
| `__proc_write` | 2 | handle: Int, data: String | `S(Maybe.None)` |
| `__proc_write_secret` | 3 | handle, secret: Secret, capability | `S(Maybe.None)` |
| `__proc_read_line` | 1 | handle | `S(String)`; `failure("process closed")` at EOF |
| `__proc_close` | 1 | handle | `S(Bool)` |
| `__proc_wait` | 1 | handle | `S(Int)` exit code |

`__proc_*` is what lets Calcium drive external backends (e.g. `dbd-sqlite`)
over a line-delimited stdio protocol. `__proc_write_secret` requires a
capability authorising sink `"proc"` with `scope.handle == handle`.

---

## 4. Conformance

A runtime satisfies this contract when the Calcium standard library behaves
identically on it. The reference suite is:

* `tests/*.test.ca` (16 files) — exercises the stdlib surface end to end.
* `tests/primitive_constraint.test.ca` — primitive argument constraints.
* `pkg/vm/primitives_test.go`, `primitives_json_test.go`, `process_test.go` —
  per-primitive behaviour, including error modes.

Two properties must hold for a new runtime:

1. **Mode fidelity.** Whatever `success`/`failure`/`Maybe`/raw shape the
   reference returns, the new runtime returns the same shape.
2. **Message fidelity is *not* required.** Error strings are diagnostic;
   matching them exactly is unnecessary, but a `failure` must not become a
   `success` or vice versa.
