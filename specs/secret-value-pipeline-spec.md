# Secret Values and Secret-Authorized Pipelines

Status: Draft
Owners: calcium-lang maintainers
Related:

- `calcium-spec.md` (language semantics)
- `specs/module-import-spec.md` (module resolution)
- `specs/module-registry-spec.md` (registry module identity)

## Summary

Calcium should provide a built-in, opaque `Secret` value that can wrap any
Calcium value. Secrets must not be printable, serializable, or passed to normal
effects. A dedicated effect pipeline operator, `!!!>`, is the only way to pass
a Secret to an effect, and it is valid only inside a capability scope. In
addition, the runtime must consult an operator-supplied module allowlist before
any Secret crosses an effect boundary.

The design has two independent gates:

1. **Source-level intent** — `Secret.with(cap) { ... }` and `!!!>`.
2. **Run-time authority** — the currently executing module must be explicitly
   allowed to perform Secret writes by the runtime policy.

Both gates must pass. Source code cannot grant itself runtime authority.

## Goals

- Prevent accidental leakage of secrets through logs, string conversion,
  interpolation, JSON/TOML serialization, database parameters, HTTP, child
  processes, and error messages.
- Make every Secret-to-effect boundary visible and easy to audit in source.
- Allow application code to delegate a Secret to a trusted library without
  revealing the raw value to that library.
- Allow the operator/embedder to decide at run time which modules may write
  secrets to effects.
- Preserve Calcium's existing effect pipeline model (`|>`, `!>`, `!?`).
- Fail closed by default.

## Non-goals

- Protecting against all malicious code or side channels (timing, memory
  inspection, process compromise).
- Guaranteed memory zeroization. Go's garbage collector makes hard guarantees
  impractical.
- Replacing OS-level secret storage or a dedicated secrets manager.
- Preventing a trusted sink from sending a secret to an attacker-controlled
  destination. Capabilities limit the destination/operation; they do not
  validate the application's business logic.

## `Secret` value

`Secret` is a built-in opaque value type. It can wrap any Calcium value, not
only strings.

```calcium
api_key: Secret = Secret.wrap("token-123");
credentials: Secret = Secret.wrap({
    user: "alice",
    password: "hunter2"
});
```

`Secret` is not an algebraic data type. It has no user-accessible constructor
pattern and no field access. Pattern matching on `Secret` is a compile error.

### Safe operations

```calcium
Secret.wrap(value)        // value -> Secret
Secret.map(secret, fn)    // apply a pure function, result stays Secret
```

`Secret.map` is the supported way to transform a secret value. The callback is
pure, and the result is wrapped again. A Secret cannot be converted back to a
plain value by ordinary language operations.

An explicit declassification operation may be added later:

```calcium
Secret.declassify(secret) // future, effectful, runtime-policy controlled
```

No `reveal` operation is included in the initial design.

### Redaction and forbidden operations

The runtime must redact Secrets everywhere a value can be rendered:

- `to_string(secret)` returns `"<secret>"`.
- `io.print` / `io.println` print `"<secret>"`.
- String interpolation renders `"<secret>"`, never the underlying value.
- Arrays/hashes containing Secrets render recursively redacted.
- REPL output and runtime error messages never include the underlying value.
- `json.stringify`, `toml.stringify`, and `dbi.quote` reject Secrets or emit a
  redacted placeholder only when an explicit redacted mode is requested.

The following operations are rejected:

- `Secret` used as a hash key (redacted keys would collide).
- `len(secret)` and ordinary string operations (`string.upper`, `concat`,
  `substring`, etc.).
- Direct equality (`secret == other`); use `Secret.equal(a, b)` when needed.
- Casting/conversion through `types.to_string` or any other generic conversion.

## Secret-authorized effect pipeline: `!!!>`

`!!!>` is a distinct operator from `!>`.

```text
|>    pure pipeline
!>    effect pipeline
!!!>  effect pipeline authorized to pass Secrets
!?    error handling
```

Example:

```calcium
Secret.with(api_cap) {
    api_key
      |> Secret.map(trim)
      !!!> http.send_secret(url);
}
```

Rules:

- A Secret passed to `!>` is a compile error when statically detectable and a
  runtime error otherwise.
- `!!!>` is valid only within an active `Secret.with(cap)` scope.
- The capability must authorize the specific effect and, where applicable, its
  destination/operation.
- `!!!>` does not authorize arbitrary effects. Each pipeline stage is checked
  independently.
- `!!!>` does not bypass the runtime module allowlist.
- Pure stages (`|>`) preserve the Secret wrapper.

Tokenization and formatting:

- The lexer recognizes `!!!>` as one token (`SECRET_PIPE`) using longest-match
  rules.
- It has the same precedence as `!>`.
- The formatter always emits `!!!>` with single spaces around it.
- `!!!x` remains three logical-not tokens; only `!!!>` is the pipeline token.

## Capabilities and `Secret.with`

A capability is an opaque value created by the application/entry environment.
It describes which effect(s) and destinations may receive Secrets.

```calcium
// Entry module only.
api_cap = Secret.capability(http.send_secret, {
    url: "https://api.example.com/upload"
});

Secret.with(api_cap) {
    token !!!> http.send_secret(url);
}
```

Capabilities are values and can be passed to libraries:

```calcium
// Library
func! upload(value: Secret, cap) =
    Secret.with(cap) {
        value !!!> http.send_secret(url);
    };

// Application
upload(token, api_cap);
```

The library can carry the Secret to an authorized sink but cannot inspect it.

### Capability creation

- Only the entry module may create capabilities.
- Imported modules cannot create capabilities, even if they declare a function
  with the same name.
- Capability creation is a built-in operation whose authority comes from the
  entry environment, not from ordinary library code.
- A capability is bound to a sink identity and optional scope (destination,
  operation, account, etc.).
- Capabilities are not forgeable by constructing plain hashes/strings with the
  same shape.

### Capability scope

`Secret.with(cap) { ... }` evaluates the block with `cap` active.

- The block is an expression.
- `!!!>` is a compile error outside any `Secret.with` block.
- The compiler may additionally check statically that the named sink is
  covered by the capability.
- The runtime verifies the capability at the effect boundary.

## Runtime module allowlist

Source-level `!!!>` expresses intent. It is not sufficient authority because a
library can write `!!!>` itself. The runtime therefore requires an independent
allowlist of modules permitted to write Secrets to effects.

```bash
calcium run \
  --allow-secret-write=entry \
  --allow-secret-write=app.auth \
  --allow-secret-write=ytnobody/http \
  main.ca
```

The allowlist can also be supplied through configuration or an embedding API.
It must never be mutable from Calcium source code.

### Policy evaluation

At a Secret-to-effect boundary, all of the following must hold:

1. The call site is a `!!!>` stage.
2. An active capability covers the effect and scope.
3. The canonical module ID of the currently executing function is allowed by
   the runtime policy.
4. The effect is a trusted sink capable of accepting a Secret.

If any check fails, the VM raises a runtime error.

### Module identity

The allowlist must use canonical module identities, not user-controlled
namespace strings.

- Local modules: resolved absolute path, optionally with a source hash.
- Registry modules: `author/name@version` from the lockfile.
- Standard library: fixed IDs such as `core.http`.
- Entry module: the special token `entry` or its canonical path.

A module must not be able to obtain authority merely by declaring:

```calcium
namespace ytnobody.http;
```

### Default policy

The default is deny. The entry module is not implicitly allowed; deployments
must opt in explicitly. For development ergonomics, the CLI may offer an
explicit `--allow-secret-write=entry` switch.

Allowlist entries are module-granular in the first version. Future versions may
extend the policy to `(module, sink, destination)` tuples.

## Trusted sinks

A trusted sink is implemented by the runtime/stdlib, not by ordinary Calcium
code. It is the only component that may unwrap a Secret.

Examples:

```calcium
http.send_secret(url, secret, cap)
db.bind_secret(stmt, index, secret, cap)
os.proc_write_secret(handle, secret, cap)
```

Pure Calcium libraries cannot declare themselves as trusted sinks. They can
accept a Secret and a capability and pass them to a trusted sink.

## Container semantics

Secrets may be nested in arrays, hashes, tuples, and ADTs. Any operation that
serializes or renders a container must recursively redact Secret leaves.

Passing a container that contains a Secret to a normal effect is subject to the
same checks as passing the Secret directly. The runtime must scan arguments
recursively (or use a taint marker propagated by containers).

## Error handling

Compile-time errors:

- Secret passed to `!>` when statically detectable.
- `!!!>` outside a `Secret.with` scope.
- `!!!>` with a sink not covered by any visible capability.
- Secret pattern matching or field/index access.
- Secret used as a hash key or in a forbidden string operation.

Runtime errors:

- `!!!>` used with a missing/expired/incorrect capability.
- Secret passed to an unapproved effect through dynamic/untyped code.
- Current module not present in the runtime allowlist.
- Trusted sink called without a valid capability.

Error messages must never contain the Secret's underlying value. They may
identify the module, effect, and capability.

## Gradual typing backstop

Calcium allows untyped code and `Any`. Compile-time checks are therefore best
effort. The VM must enforce Secret safety at effect boundaries regardless of
static types.

## Compatibility

- `!!!>` was not previously a valid operator, so adding it is backward
  compatible.
- `Secret` becomes a reserved built-in type name.
- Existing values are unaffected; wrapping is explicit.
- Existing effect pipelines continue to work for non-Secret values.
- JSON/DB modules are updated to reject or redact Secret values by default.

## Test plan

Compile-time:

- `secret !> io.println;` fails.
- `secret !!!> io.println;` outside `Secret.with` fails.
- `Secret` pattern matching fails.
- Secret as hash key fails.
- `Secret` serialization fails.
- `!!!x` still parses as logical-not tokens.

Runtime:

- Redaction in `to_string`, print, interpolation, arrays, hashes, errors.
- Authorized `Secret.with(cap) { secret !!!> sink; }` succeeds when module is
  allowed.
- Same code fails when module is not in the runtime allowlist.
- A library cannot create a capability.
- A library can forward a Secret to a trusted sink with a delegated capability.
- Dynamic/untyped bypass attempts fail at the effect boundary.
- Nested Secrets in containers are detected.
- JSON/TOML/DB rejection paths.

## Implementation phases

1. `Secret` value, redaction, `Secret.wrap`, `Secret.map`.
2. `!!!>` token/parser/formatter and static Secret-in-`!>` checks.
3. `Secret.with` capability scopes and capability values.
4. Trusted sink primitives.
5. Runtime module allowlist and module identity propagation.
6. JSON/TOML/DB integration and end-to-end tests.

Security-sensitive behavior should be reviewed before phase 5 ships; phases 1–2
are useful as an accidental-leak prevention layer, but not as full runtime
authorization.
