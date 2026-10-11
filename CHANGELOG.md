# Changelog

## Unreleased

### Breaking changes

- **`len` / `char_at` / `substring` on strings now use rune (Unicode code point) semantics instead of raw byte semantics.**
  - `len("aアb😀")` now returns `4`, not the byte length `1+3+1+4=9`.
  - `string.char_at` / `string.substring` index multi-byte characters by code point, not byte offset.
  - This unifies the string API with `string.length` and the `__str_*` primitives. Most code is unaffected; only code that indexed/counted multi-byte strings by byte position may change behavior.

### Features

- String literals now support `\b`, `\f`, and `\uXXXX` escapes. `\uXXXX` decodes a BMP code point or a surrogate pair (`\uD83D\uDE00` → `😀`), matching the JSON module.

### Standard library

- Added `string.from_code(cp)`: converts a Unicode code point to its string representation (returns `success`/`failure`; rejects surrogates and out-of-range values).

### Language

- Added built-in common constraints available everywhere (as parameter annotations and `?` checks): `NonZero`, `Positive`, `NonNegative`, `Natural`, `Finite`, `NonEmpty`, `Empty`, `Digits`, `Lowercase`, `Uppercase`. Statically-detectable violations on literal call arguments are caught at compile time. User-defined constraints with the same name shadow the built-ins.

---

## How releases are tagged

Releases are tagged automatically on merge to `main` (patch bump) and released via GoReleaser. For the rune-semantics change, code that relied on the byte length of multi-byte strings must be reviewed before upgrading past the accompanying release.