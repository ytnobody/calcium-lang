# Bytecode Compilation Specification

## Overview

Pre-compile Calcium source files (.ca) to bytecode files (.bone) to skip parsing and compilation at runtime.

## Benefits

1. **Source obfuscation during distribution** - Distribute programs without exposing source code
2. **Faster startup time** - Skip parsing and compilation
3. **Pre-optimization preservation** - Regex compilation results etc. are also saved

---

## Commands

### Compile

```bash
# Basic form (output is hello.bone)
calcium compile hello.ca

# Specify output file
calcium compile hello.ca -o output.bone
```

### Execute

```bash
# Run source directly
calcium hello.ca

# Run compiled bytecode
calcium hello.bone

# Can also use run command
calcium run hello.ca
calcium run hello.bone
```

---

## File Format

### .bone (Calcium Bytecode) Format

```
+-------------------+
| Magic: 4 bytes    |  "CALB"
+-------------------+
| Version: 2 bytes  |  major, minor
+-------------------+
| NumConstants: 4   |  Number of constants (uint32, big-endian)
+-------------------+
| Constants...      |  Serialized data for each constant
+-------------------+
| InsLen: 4 bytes   |  Instruction length (uint32)
+-------------------+
| Instructions...   |  Bytecode instructions
+-------------------+
```

### Constant Serialization

Each constant consists of type tag (1 byte) + data:

| Type Tag | Type | Data Format |
|----------|------|-------------|
| 0 | null | None |
| 1 | bool | 1 byte (0 or 1) |
| 2 | int | 8 bytes (int64, big-endian) |
| 3 | float | 8 bytes (IEEE 754) |
| 4 | string | 4 byte length + UTF-8 data |
| 5 | function | See below |
| 6 | regex | See below |

### Function Serialization

```
[Name: string]
[NumParams: uint32]
[Params: string[]]
[NumLocals: uint32]
[IsEffect: byte]
[BodyLen: uint32]
[Body: bytes]
```

### Regex Serialization

```
[Pattern: string]
[Flags: string]
```

*Recompiled with `regexp.Compile()` on load

---

## Version Compatibility

- Different major version: cannot load
- Different minor version: allowed (backward compatible)

Current version: **1.0**

---

## Implementation Files

| File | Content |
|------|---------|
| `pkg/bytecode/serialize.go` | Serialization/deserialization implementation |
| `pkg/bytecode/serialize_test.go` | Tests |
| `cmd/calcium/main.go` | CLI compile/run commands |

---

## Limitations

### Non-serializable Types

The following types are not in the constant pool, so no issues:
- Closure (created at runtime)
- Module (loaded at runtime)
- Task/Handler/EventSource (for async processing, created at runtime)

### Dynamic Features

- Modules loaded via `use` are resolved at runtime
- Module functions are bound at runtime

---

## Usage Examples

### Basic Workflow

```bash
# Development: run source directly
calcium app.ca

# Release: compile and distribute
calcium compile app.ca -o app.bone

# User: run compiled
calcium app.bone
```

### Regex Optimization

```calcium
// app.ca
use core.regex;
pattern = /^[a-z]+@[a-z]+\.[a-z]{2,}$/i;
email = "test@example.com";
regex.matches(email, pattern) |> io.println;
```

Regex is compiled at compile time and saved in `.bone` file.
At runtime, pre-compiled regex is immediately usable.

---

## Future Extensions (Under Consideration)

1. **Debug info** - Store line number mappings
2. **Compression** - Reduce file size with gzip etc.
3. **Signature** - Signature for tampering detection
4. **Optimization levels** - Options like `-O0`, `-O1`, `-O2`
