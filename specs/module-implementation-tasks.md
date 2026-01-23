# Module System Implementation Tasks

## Overview

Based on the specification, the following implementations are needed.

---

## 1. Registry Operations Tools (GitHub Actions)

### validate.yml - PR Validation

```yaml
# .github/workflows/validate.yml
on:
  pull_request:
    paths: ['packages/**']
```

**Implementation:**
- [ ] TOML format check (meta.toml, {version}.toml)
- [ ] Required fields validation
- [ ] URL reachability verification (base_url + each file)
- [ ] SHA256 validation (fetch actual file → verify hash)
- [ ] License validation (unspecified → auto-apply MIT)
- [ ] AI review invocation

### build-index.yml - Index Rebuild

```yaml
# .github/workflows/build-index.yml
on:
  push:
    branches: [main]
    paths: ['packages/**']
```

**Implementation:**
- [ ] Traverse packages/
- [ ] Generate index/all.toml
- [ ] Generate index/{A-Z}.toml
- [ ] Update author.toml modules list

---

## 2. Registry Scripts

### scripts/validate.sh

```bash
#!/bin/bash
# Validate TOMLs added/changed in PR
```

**Features:**
- TOML parsing (tomlq or Go tool)
- HTTP requests (curl)
- SHA256 calculation (sha256sum)

### scripts/build-index.sh

```bash
#!/bin/bash
# Rebuild index from packages/
```

### scripts/ai-review.sh

```bash
#!/bin/bash
# Call AI API for review
```

---

## 3. Calcium CLI Extension

### calcium cache

```
calcium cache <file.ca>           # Pre-fetch dependencies
calcium cache --info              # Cache info
calcium cache --clear             # Clear cache
```

**Implementation (Go):**
- [ ] Parse `use "https://..."` statements
- [ ] Fetch metadata TOML
- [ ] Get hashes from file list
- [ ] Download actual files & verify hashes
- [ ] Save to `~/.calcium/cache/`

### calcium run extension

```
calcium run --import-map=calcium.imports.toml src/main.ca
calcium run --lock=calcium.lock src/main.ca
calcium run --cached-only src/main.ca
```

**Implementation (Go):**
- [ ] Load import map & URL conversion
- [ ] Load lock file & verify hashes
- [ ] --cached-only disables network

---

## 4. Required Libraries/Modules

### Go Side (CLI)

| Package | Purpose |
|---------|---------|
| `github.com/BurntSushi/toml` | TOML parsing |
| `crypto/sha256` | Hash calculation |
| `net/http` | HTTP client |
| `os` | Cache directory management |

### Shell Script Side (CI)

| Tool | Purpose |
|------|---------|
| `tomlq` or `yj` | TOML parsing (jq-like) |
| `curl` | HTTP requests |
| `sha256sum` | Hash calculation |
| `gh` | GitHub CLI (PR operations) |

---

## 5. Implementation Order

### Phase 1: Minimal Registry

1. Create GitHub repository `calcium-lang/modules`
2. Setup directory structure
3. Implement `scripts/validate.sh`
4. Configure `validate.yml`
5. Register sample module (manual)

### Phase 2: Calcium CLI Support

1. Add TOML parser (Go)
2. Parser support for `use "https://..."`
3. Implement HTTP client
4. Implement cache management
5. Implement `calcium cache` command

### Phase 3: Automation

1. Implement `build-index.yml`
2. AI review integration
3. `calcium run` option extension
4. Lock file generation

### Phase 4: Usability Improvements

1. `calcium search` command
2. `calcium publish` command (metadata generation helper)
3. Error message improvements
4. Documentation

---

## 6. File Placement (Planned)

```
calcium/
├── cmd/calcium/
│   └── main.go              # CLI entry (cache, run extension)
├── pkg/
│   ├── module/
│   │   ├── resolver.go      # Module resolution
│   │   ├── cache.go         # Cache management
│   │   ├── fetch.go         # HTTP fetch + hash verification
│   │   └── toml.go          # TOML parsing
│   └── ...
└── specs/
    ├── module-import-spec.md
    └── module-registry-spec.md
```

---

## 7. Test Plan

### Unit Tests

- TOML parsing
- SHA256 calculation
- URL → cache path conversion
- Import Map application

### Integration Tests

- Metadata fetch → file fetch → hash verify → cache save
- Lock file generation/loading
- Execution with --cached-only

### E2E Tests

- Fetch module from actual registry
- PR validation workflow
