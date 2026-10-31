# `.bone` Bytecode Format Specification

Status: **normative**. This document is the contract between independent Calcium
implementations. The Go implementation in `pkg/bytecode` is the reference; if
this document and the code disagree, that is a bug in one of them.

The format is produced by `calcium compile` and consumed by `calcium run`,
`calcium <file>.bone`, and the `calcium build` standalone packaging.

---

## 1. Compatibility rules

Two independent rules keep older readers usable:

1. **Opcode numbers are append-only.** `OpCode` values are written into files.
   A new opcode must be appended after the highest existing value; inserting
   or reordering opcodes silently changes what existing files mean. A reader
   that meets an unknown opcode must fail with `unknown opcode: N` rather than
   guessing.
2. **Structural changes bump the minor version.** A reader accepts any file
   whose major version is `<= VersionMajor`, and interprets minor-versioned
   optional fields only when the file's minor version is high enough (see
   §4.1). Structural changes and new optional fields bump `VersionMinor`;
   appending an opcode does not, because opcodes are append-only by rule 1 and
   a reader rejects an unknown one by number.

Current version: **1.1**.

| Constant | Value | Notes |
|---|---|---|
| Magic | `CALB` (0x43 0x41 0x4C 0x42) | "CALcium Bytecode" |
| `VersionMajor` | 1 | reader rejects a greater major |
| `VersionMinor` | 1 | 1.1 added the `ModuleID` field to function records |
| File extension | `.bone` | |

---

## 2. Scalar encodings

All multi-byte integers in the file body are **big-endian**.

| Name | Encoding |
|---|---|
| `u8` | 1 unsigned byte |
| `u32` | 4 bytes, big-endian unsigned |
| `i64` | 8 bytes, big-endian two's complement |
| `f64` | 8 bytes, big-endian IEEE 754 double |
| `string` | `u32` byte-length, then that many UTF-8 bytes (no terminator, no NUL) |
| `bool` | `u8`: `0` = false, non-zero = true |

The standalone-executable packaging in §7 uses **little-endian** for its
trailer. That is a separate envelope and is unrelated to the file body.

---

## 3. File layout

```
+----------------------------------+
| magic            "CALB"   (4)    |
| version_major    u8              |
| version_minor    u8              |
| constant_count   u32             |
| constants        constant[constant_count] |
| ins_len          u32             |
| instructions     u8[ins_len]     |
+----------------------------------+
```

`ins_len` counts bytes, not instructions.

### 3.1 Reader limits

A reader must reject input that cannot be satisfied by the remaining bytes, and
must enforce these caps before allocating:

| Limit | Value | Applies to |
|---|---|---|
| `maxBytecodeElements` | 1,048,576 (1 << 20) | constant count, parameter count, ADT value count |
| `maxBytecodeStringLen` | 67,108,864 (64 MiB) | string length, instruction length |

For every count and length, the value must also be `<=` the number of bytes
remaining in the input. This is what stops a 4-byte length from triggering a
huge allocation.

---

## 4. Constants

Each constant is a `u8` type tag followed by a payload.

| Tag | Name | Payload |
|---|---|---|
| 0 | `LegacyNone` | none. Decoded as `Maybe.None`, and **never written**. Kept so that files produced before the `Maybe` change still load. |
| 1 | `Bool` | `u8` |
| 2 | `Int` | `i64` |
| 3 | `Float` | `f64` |
| 4 | `String` | `string` |
| 5 | `Function` | function record (§4.1) |
| 6 | `Regex` | regex record (§4.2) |
| 7 | `ADT` | ADT record (§4.3) |

Any other tag is an error (`unknown constant type tag: N`).

Only these constant kinds may appear in a file. In particular arrays, hashes,
tuples and closures are built at run time by instructions and are not
serialisable constants; a writer that needs one must reject it.

### 4.1 Function record

```
name         string
module_id    string          (only when version_minor >= 1)
param_count  u32
params       string[param_count]
num_locals   u32
is_effect    u8              (1 = `func!`)
body_len     u32
body         u8[body_len]
```

A 1.0 file has no `module_id` field; readers must supply the default
(`"entry"`) in that case. Functions nest arbitrarily: a function constant's
`body` may reference further function constants.

### 4.2 Regex record

```
pattern  string     ; the source pattern, as written between the slashes
flags    string     ; subset of "ims"
```

The pattern is stored in Calcium form, not Go form. A reader reconstructs the
native regex from `pattern` and `flags`; the Go reference implementation
prefixes `(?flags)` when flags are present. Because the pattern is source text,
readers must be able to compile it (or defer the error to first use — the
reference implementation compiles eagerly and fails the load).

### 4.3 ADT record

```
type_name   string
tag         string
value_count u32
values      constant[value_count]
```

`values` are recursively encoded constants, so an ADT payload may contain
other ADTs, ints, strings, functions and regexes.

---

## 5. Instruction encoding

An instruction is a `u8` opcode followed by its operands. Operand widths are
fixed per opcode (table below) and are **not** self-describing, so a reader
must consult the opcode table to walk the stream.

Operands that are indices (constants, globals, locals, builtins, module paths)
are unsigned. Jump operands are absolute instruction offsets, not relative.
Operand widths are in bytes.

A disassembler must show the instruction length as part of the decode, since
some opcodes (`OpClosure`, `OpConstructADT`, `OpMatchADT`) carry two operands.

### 5.1 Opcode table

`()` = no operands. `u16` = 2-byte big-endian. `u8` = 1 byte.

| # | Opcode | Operands |
|---|---|---|
| 0 | `OpConstant` | u16 constant index |
| 1 | `OpPop` | () |
| 2 | `OpDup` | () |
| 3 | `OpGetLocal` | u8 |
| 4 | `OpSetLocal` | u8 |
| 5 | `OpGetGlobal` | u16 |
| 6 | `OpSetGlobal` | u16 |
| 7 | `OpAdd` | () |
| 8 | `OpSub` | () |
| 9 | `OpMul` | () |
| 10 | `OpDiv` | () |
| 11 | `OpMod` | () |
| 12 | `OpPow` | () |
| 13 | `OpNeg` | () |
| 14 | `OpEqual` | () |
| 15 | `OpNotEqual` | () |
| 16 | `OpLessThan` | () |
| 17 | `OpGreaterThan` | () |
| 18 | `OpLessEqual` | () |
| 19 | `OpGreaterEqual` | () |
| 20 | `OpNot` | () |
| 21 | `OpAnd` | u16 jump offset |
| 22 | `OpOr` | u16 jump offset |
| 23 | `OpJump` | u16 jump offset |
| 24 | `OpJumpIfFalse` | u16 jump offset |
| 25 | `OpJumpIfTrue` | u16 jump offset |
| 26 | `OpCall` | u8 argument count |
| 27 | `OpReturn` | () |
| 28 | `OpClosure` | u16 function constant index, u8 free-variable count |
| 29 | `OpGetFree` | u8 |
| 30 | `OpCallEffect` | u8 argument count |
| 31 | `OpArray` | u16 element count |
| 32 | `OpArrayConcat` | u16 element count |
| 33 | `OpIndex` | () |
| 34 | `OpSpread` | () |
| 35 | `OpCallSpread` | () |
| 36 | `OpTuple` | u16 element count |
| 37 | `OpHash` | u16 pair count |
| 38 | `OpGetMember` | u16 property-name constant index |
| 39 | `OpWrapSuccess` | () |
| 40 | `OpWrapFailure` | () |
| 41 | `OpUnwrapOrJump` | u16 jump offset |
| 42 | `OpIsSuccess` | () |
| 43 | `OpIsFailure` | () |
| 44 | `OpUnwrap` | () |
| 45 | `OpBuiltin` | u8 builtin index |
| 46 | `OpCheckConstraint` | () |
| 47 | `OpLoadModule` | u16 module-path constant index |
| 48 | `OpNone` | () |
| 49 | `OpTrue` | () |
| 50 | `OpFalse` | () |
| 51 | `OpStayBegin` | u16 number of state pairs |
| 52 | `OpStayEnd` | () |
| 53 | `OpStayGetState` | u16 state-key constant index |
| 54 | `OpContinue` | u16 number of state pairs to update |
| 55 | `OpLeave` | () |
| 56 | `OpSpawn` | () |
| 57 | `OpExpects` | () |
| 58 | `OpHandlerReady` | () |
| 59 | `OpHandlerReset` | () |
| 60 | `OpHandlerPause` | () |
| 61 | `OpHandlerResume` | () |
| 62 | `OpCancel` | () |
| 63 | `OpAll` | () |
| 64 | `OpTimeout` | () |
| 65 | `OpInterval` | () |
| 66 | `OpTailCall` | u8 argument count |
| 67 | `OpConstructADT` | u16 tag-name constant index, u8 arity |
| 68 | `OpMatchADT` | u16 tag-name constant index, u8 arity |
| 69 | `OpConstructMaybeSome` | () |
| 70 | `OpAddOverload` | () |
| 71 | `OpRegexMatch` | () |
| 72 | `OpSecretWrap` | () |
| 73 | `OpSecretMap` | () |
| 74 | `OpSecretCapability` | () |
| 75 | `OpSecretWithBegin` | () |
| 76 | `OpSecretWithEnd` | () |
| 77 | `OpCheckSecretCapability` | u16 sink-name constant index |
| 78 | `OpAllowSecretCall` | () |
| 79 | `OpBitAnd` | () |
| 80 | `OpBitOr` | () |
| 81 | `OpBitXor` | () |
| 82 | `OpBitNot` | () |
| 83 | `OpShl` | () |
| 84 | `OpShr` | () |
| 85 | `OpMatchFail` | () |

**The next opcode is 86.** Append only.

### 5.2 Opcode numbering history

Bitwise opcodes 79–84 were added after the initial release. They were appended
rather than placed next to `OpAdd` precisely to preserve rule (1) in §1; older
readers reject them by number, which is the intended behaviour for a file that
uses a newer language feature.

`OpMatchFail` (85) was added when match arms became fully conditional: it pops
the subject and raises a runtime error when no arm matched. It is emitted only
when a match expression's final arm can reject, so bytecode for a match ending
in `_` is only affected by the removal of an implicit catch-all.

---

## 6. Bytecode-level optimisations

`-O2` runs peephole and CSE passes over the instruction stream before writing.
Optimisations are never required for correctness and never change the observable
result; a reader must not assume a particular instruction sequence for a given
source program. Opcode-level identity between two compilers is therefore *not*
part of this contract — only program behaviour is (see the differential-testing
note in `selfhost-scope.md`).

---

## 7. Standalone executable packaging (`calcium build`)

`calcium build` emits a runnable program by concatenating the interpreter with
the bytecode:

```
+-----------------------------+
| interpreter binary          |   (the `calcium` executable, itself stripped)
| bone payload                |   (a complete file per §3)
| bone_len           u64 LE   |   8 bytes, little-endian
| "BONEPACK"                 |   8 ASCII bytes
+-----------------------------+
```

Notes:

* **The trailer is little-endian**, unlike the rest of the format.
* The magic is the last 8 bytes of the file. A runtime reads `bone_len` from
  the 8 bytes before the magic, so the payload is
  `[len - 16 - bone_len, len - 16)`.
* Before appending, an existing trailer is stripped, so building a binary that
  already embeds a program does not nest payloads.
* The output file is made executable (`0755`).
* This envelope is inherently implementation-specific: a self-hosted runtime may
  package its own interpreter instead. Only the embedded payload must follow
  §3.

---

## 8. Conformance checklist

A reader is conformant when it:

1. Rejects a magic mismatch and an unsupported major version.
2. Accepts minor version 0 and 1 (supplying `module_id = "entry"` for 0).
3. Decodes `LegacyNone` as `Maybe.None`.
4. Enforces both caps in §3.1 *and* the remaining-bytes check.
5. Reports an unknown opcode by number instead of panicking.
6. Walks instructions using the widths in §5.1, including multi-operand
   instructions.

A writer is conformant when it:

1. Emits `CALB` + 1.1.
2. Never emits `LegacyNone`.
3. Never emits a constant kind that has no tag in §4.
4. Uses big-endian for all body integers.
5. Appends new opcodes rather than inserting them.

---

## 9. Reference implementation

| Concern | Location |
|---|---|
| Encode/decode | `pkg/bytecode/serialize.go` |
| Opcode numbers, operand widths | `pkg/bytecode/bytecode.go` (`definitions`, `Make`, `ReadOperands`) |
| Standalone packaging | `cmd/calcium/main.go` (`packMagic`, `stripEmbeddedData`) |
| Round-trip tests | `pkg/bytecode/serialize_test.go` |
