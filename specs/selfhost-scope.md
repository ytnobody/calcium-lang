# Self-Hosting Scope

Status: **agreed scope** for reimplementing Calcium in Calcium. This document
records what is in scope, what stays native, and how progress is verified. It is
the contract the `calcium-selfhost` repository works against.

Companion documents:

* `bone-format.md` — the serialised program format both implementations share.
* `ffi-primitives.md` — the native boundary that is never reimplemented.

---

## 1. Scope

In scope: the components that turn source text into executed behaviour, and the
developer tooling that sits directly on top of the same front end.

| Group | Components | Go LOC | Notes |
|---|---|---|---|
| Front end | `token`, `lexer`, `ast`, `types`, `parser`, `typechecker` | 4,192 | Includes the gradual type system |
| Compiler | `compiler`, `static_constraint`, `optimizer`, `bytecode` | 6,190 | Emits the `bone-format.md` format |
| Linter | `linter` | 164 | Depends only on the front end + compiler |
| Formatter | `formatter` | 556 | Depends only on the front end |
| VM core | `vm.go`, `stdlib.go`, `options`, `cache` | 5,451 | Interpreter loop, frames, builtins, async runtime |
| **Total** | | **16,553** | Expect 1.5–2× in Calcium, i.e. 25k–33k lines |

### 1.1 Dependency reality

```
token → lexer → ast → parser → typechecker ─┐
                                            ├→ compiler → bytecode
                       optimizer ───────────┘
formatter ← lexer/parser/ast
linter    ← lexer/parser/typechecker/compiler
vm        ← ast, bytecode, compiler, lexer, optimizer, parser, value
```

The compiler cannot be self-hosted without the front end: the units are
inseparable. Formatter and linter come almost free once the parser exists, which
is why they are included.

## 2. Out of scope

| Component | Go LOC | Rationale |
|---|---|---|
| Native boundary (`primitives`, `primitives_json`, `process`, `builtin`) | 2,702 | See `ffi-primitives.md`. Cannot be written in Calcium by definition. |
| `value` (value model) | 954 | Does not get ported — see §3. |
| `lsp` | 1,042 | Depends only on `lexer/parser/ast/token`; a good later target, not required |
| `repl` | 482 | Thin wrapper over the VM |
| `cmd/calcium`, `cmd/bone` | 1,901 | CLI plumbing |
| `bone` (package manager) | 1,327 | Orthogonal to the language |

## 3. `value.go` does not get ported

The 23 `TYPE_*` variants are the Go representation of Calcium values. In a
self-hosted runtime, values *are* Calcium values and `value.go` disappears.

Exception: `TYPE_TASK`, `TYPE_CHANNEL`, `TYPE_EVENT_SOURCE` and `TYPE_SECRET*`
wrap host pointers (goroutines, channels). These stay as **opaque handles**
passed through the primitive layer. Deciding which of the 23 types stay opaque
is part of designing the VM's value representation and must be settled before
the VM work starts.

## 4. Stages

### SH-1 — Prerequisites (in the Go repository)

Small changes to the reference implementation that make it a usable host:

| Item | Status |
|---|---|
| Call-frame exhaustion reports an error instead of panicking (was a Go panic at depth 8,192) | done |
| Stack traces truncated with an omission marker | done |
| Bitwise operators `& \| ^ << >>`, prefix `^` | done |
| `.bone` format frozen and documented (`bone-format.md`) | done |
| Native boundary documented (`ffi-primitives.md`) | done |

Deliberately **not** done: a mutable-buffer primitive. Measurement showed
immutable cons lists are already O(n) (`100k` build + reverse = 139 ms), and the
standard library does not use `push` at all. `push` and `hash_set` remain O(n),
which is a real constraint for symbol tables; revisit only when a stage proves
it necessary.

### SH-2 — Front end

Implement lexer → parser → ast in Calcium, then `typechecker`.

### SH-3 — Compiler

A Calcium compiler that emits the `bone-format.md` format, executed by the
native Go VM. **This is the point at which self-hosting is achieved for
practical purposes.**

### SH-4 — VM

Reimplement the bytecode interpreter in Calcium. This runs on the Go VM, so it
is double interpretation; expect 10–50× slowdown. Treat SH-4 as a stretch goal
and re-evaluate after SH-3.

## 5. Verification

The existing Go test suite is 22,549 lines and is the acceptance oracle.

```
reference (Go, pinned) --compile--> a.bone
candidate            --compile--> b.bone
run both on the shared corpus and compare behaviour
```

Corpus: `examples/`, `tests/` (16 `.test.ca` files), `csv/`, `test/`.

Rules:

* **Behaviour only, not byte-identical output.** Optimisation passes are free to
  differ; `-O2` instruction streams are not part of the contract
  (`bone-format.md` §6).
* **The oracle is pinned to a commit of the Go implementation**, not to a
  release, so that semantic changes are picked up deliberately rather than weeks
  later.
* The corpus is consumed from the Go repository (submodule or CI checkout) and is
  never copied.

### 5.1 Semantic churn is the main risk

Grammar has been stable (1 lexer change in the last 30 commits) but semantics
have not. The current unreleased changes alone include `Null` → `Maybe`,
byte → rune string semantics, `array.find` returning `Maybe`, and composite
constraint annotations. These are compiler/typechecker changes, i.e. exactly the
code being reimplemented.

The mitigation is the pinning above: a semantic change becomes "bump the pin,
the differential test goes red, fix it" — a mechanical loop.

## 6. Repository layout

The self-hosted implementation lives in **its own repository**,
`calcium-selfhost`. Rationale: the Go repository becomes the reference
implementation, the two have different lifecycles, and the switchover at SH-3+
should be a visible event.

The cost of separation is supply-chain synchronisation, which is handled by
pinning and by keeping all shared contracts in *this* repository:

```
calcium-lang/                  (reference implementation, source of truth)
  pkg/                         Go implementation
  specs/bone-format.md         shared format contract
  specs/ffi-primitives.md      shared native boundary
  specs/selfhost-scope.md      this document
  examples/  tests/  csv/      differential-test corpus

calcium-selfhost/              (candidate implementation)
  meta.toml                    bone module definition
  lexer/ parser/ compiler/ formatter/ linter/ vm/    *.ca
  tests/                       differential test harness
  .github/workflows/ci.yml     builds the oracle from calcium-lang@<sha>
```

Contracts and the corpus live in the reference repository because both
implementations must obey them; duplicating them in the candidate repository
would create exactly the drift that separate repositories make dangerous.

## 7. Open decisions

1. **Value representation.** Which `TYPE_*` variants become opaque handles
   (§3). Must be settled before SH-4 and informs SH-2/SH-3.
2. **Async runtime.** The VM has ~35 `go func`/`chan` sites. Calcium's own async
   is implemented on top of that runtime, so a Calcium VM would nest async
   inside async. Decide explicitly whether the scheduler stays native.
3. **LSP timing.** `lsp` depends only on the front end, so it is nearly free
   after SH-2 and is excellent dogfooding. Undecided whether to pull it into
   scope.
