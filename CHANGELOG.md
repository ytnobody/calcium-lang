# Changelog

## Unreleased

### Breaking changes

- **Lookup absence now uses the built-in `Maybe` type.** `get(collection, key)`, `head(array)`, and `string.char_at(s, index)` return `Maybe.Some(value)` or `Maybe.None`; use `get(collection, key, default)` when a raw default is desired. Missing hash keys and out-of-range array/tuple/ADT indices now raise runtime errors instead of returning `null`.
- **Removed the user-visible `Null` type and null APIs.** `types.null`, `Null` type annotations, `Null?`, `types.is_null`, `assert.is_null`, and `assert.not_null` are gone. Use `Maybe`, `Maybe.None`, `assert.is_none`, and `assert.is_some` instead.
- **External nulls map to `Maybe.None`.** JSON `null`, database `NULL`, and TOML absence now become `Maybe.None`; `json.stringify(Maybe.None)` emits `null`, and `json.stringify(Maybe.Some(v))` serializes `v`. `dbi.query_row` returns `success(Maybe.None)` when no row matches, and `dbi.quote(Maybe.None)` returns `NULL`.
- **Effect functions with no useful value return `success(Maybe.None)`** instead of `success(null)` (print, file/directory mutations, channel send/close, async control, etc.).
- **`len` / `char_at` / `substring` on strings now use rune (Unicode code point) semantics instead of raw byte semantics.**
  - `len("aアb😀")` now returns `4`, not the byte length `1+3+1+4=9`.
  - `string.char_at` / `string.substring` index multi-byte characters by code point, not byte offset.
  - This unifies the string API with `string.length` and the `__str_*` primitives. Most code is unaffected; only code that indexed/counted multi-byte strings by byte position may change behavior.

### Features

- Added the built-in **`Maybe`** optional-value type with qualified constructors `Maybe.Some(value)` and `Maybe.None`, usable in expressions, pattern matches, type annotations (`x: Maybe`), and higher-order functions (`map(Maybe.Some)`).
- String literals now support `\b`, `\f`, and `\uXXXX` escapes. `\uXXXX` decodes a BMP code point or a surrogate pair (`\uD83D\uDE00` → `😀`), matching the JSON module.

### Standard library

- Added `string.from_code(cp)`: converts a Unicode code point to its string representation (returns `success`/`failure`; rejects surrogates and out-of-range values).

### Language

- Added built-in common constraints available everywhere (as parameter annotations and `?` checks): `NonZero`, `Positive`, `NonNegative`, `Natural`, `Finite`, `NonEmpty`, `Empty`, `Digits`, `Lowercase`, `Uppercase`. Statically-detectable violations on literal call arguments are caught at compile time. User-defined constraints with the same name shadow the built-ins.
- Added composite parameter annotations: `func f(a: Positive & NonZero)` (AND) and `func f(a: Int | String)` (UNION), which expand to synthetic constraints with static + runtime enforcement. Mixing `&` and `|` in one annotation is a parse error.
- Static analysis now also validates **return constraints**: a function whose literal (or constant-folded) return value violates its return constraint is a compile error.
- Static analysis now applies **constant propagation**: a variable assigned a literal is known to the checker, so `x = 0; f(x)` compiled against a `: Positive` param is a compile error (rather than only a runtime failure).
- Static analysis now resolves simple **function aliases** (`g = f`), so `g(0)` is checked against `f`'s constraints.

### Features

- Added a standalone **linter** command `calcium lint [--json] <file.ca> [...]`, backed by the new `pkg/linter` package. It performs static analysis of Calcium source without executing it, reporting parse errors, gradual-type mismatches, and static constraint violations (including constant propagation, function aliases, and return constraints). Exits `1` when findings are reported, making it suitable for CI. The CI lint job now also runs `calcium lint` over well-formed example sources.
- Added generic **external process execution** to `core.os`: `os.proc_start`, `os.proc_write`, `os.proc_read_line`, `os.proc_close`, and `os.proc_wait` let Calcium launch and talk to long-lived external programs over a stdio text stream (used by driver backends such as `dbd-sqlite` speaking a line-delimited JSON protocol). Spawned children are tracked and terminated when the owning program/REPL exits or on `os.exit`, so they never outlive their parent. This is a general OS facility and brings no database-specific code or new dependencies into the core.

### CI / Releases

- Fixed release publishing: tags pushed by the auto-release workflow (via the repository `GITHUB_TOKEN`) never triggered the GoReleaser `Release` workflow, so GitHub Releases stalled at `v0.2.0` while git tags advanced to `v0.2.21`. The auto-release workflow now explicitly dispatches `Release` (`workflow_dispatch`, an allowed exception to the GITHUB_TOKEN rule) for each new tag.
- The `Release` workflow now accepts a manual `tag` input (via `workflow_dispatch`) so any tag can be published on demand.
- Added a `Backfill Releases` workflow that publishes a GitHub Release for every existing `vX.Y.Z` tag that lacks one, and a scheduled `Release Sync Check` that fails if the latest tag ever has no matching Release again.

---

## How releases are tagged

Releases are tagged automatically on merge to `main` (patch bump) and released via GoReleaser. For the rune-semantics change, code that relied on the byte length of multi-byte strings must be reviewed before upgrading past the accompanying release.