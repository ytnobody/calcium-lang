# Calcium Module System Specification

## Status: Implemented

Module import and dependency management with bone package manager and Boneyard registry.

---

## Overview

Calcium supports three types of module imports:

1. **Standard library** - Built-in modules (`core.*`)
2. **Local modules** - Project-local files
3. **External modules** - From Boneyard registry or GitHub

---

## Import Syntax

### Standard Library

```calcium
use core.io!;      // Effect module
use core.math;     // Pure module
use core.string;
use core.array;
```

### Local Modules

```calcium
use mymodule;           // ./mymodule/mod.ca or ./mymodule.ca
use utils.helper;       // ./utils/helper/mod.ca or ./utils/helper.ca
```

### External Modules

```calcium
// Format 1: author/module (recommended)
use ytnobody/json;
use ytnobody/json!;     // Effect module

// Format 2: GitHub URL
use "github.com/ytnobody/json-calcium";
use "github.com/ytnobody/json-calcium"!;
```

---

## Package Manager (bone)

### Installation

```bash
go build -o bone ./cmd/bone
```

### Commands

```bash
# Initialize a new project
bone init [name]

# Add modules
bone add author/module              # Local install (calcium_modules/)
bone add --global author/module     # Global install (~/.calcium/cache/)
bone add author/module@1.0.0        # Specific version

# Manage modules
bone list                           # List installed modules
bone update [module]                # Update modules
bone remove author/module           # Remove a module

# Configuration
bone config                         # Show all config
bone config get registry_url        # Get config value
bone config set registry_url URL    # Set config value
```

### Project Structure

```
my-project/
├── meta.toml           # Project metadata
├── calcium.lock        # Lock file (versions)
├── calcium_modules/    # Local modules
│   └── author/
│       └── module/
│           └── mod.ca
└── main.ca             # Entry point
```

### meta.toml

```toml
name = "my-project"
author = "yourname"
description = "My Calcium project"
license = "MIT"
keywords = ["web", "utility"]
entry = "main.ca"

[dependencies]
"ytnobody/json" = "^1.0.0"
"author/module" = ">=2.0.0"
```

### calcium.lock

Auto-generated lock file with exact versions:

```toml
[[modules]]
  name = "json"
  author = "ytnobody"
  version = "1.0.0"
  commit = "abc123def456..."

[[modules]]
  name = "module"
  author = "author"
  version = "2.1.0"
  commit = "def456abc789..."
```

### Configuration

Stored in `~/.calcium/config.toml`:

```toml
registry_url = "https://raw.githubusercontent.com/ytnobody/boneyard/main/index"
```

---

## Module Resolution

### Resolution Order

When loading `use author/module`:

1. **In-memory cache** - Already loaded modules
2. **Local project** - `calcium_modules/author/module/mod.ca`
3. **Global cache** - `~/.calcium/cache/author/module/mod.ca`
4. **Remote fetch** - GitHub (saved to global cache)

### Directory Structure

```
# Project local
./calcium_modules/
└── author/
    └── module/
        └── mod.ca

# Global cache
~/.calcium/
├── config.toml
└── cache/
    └── author/
        └── module/
            └── mod.ca
```

### GitHub URL Resolution

For modules not in Boneyard, tries:
1. `github.com/{author}/{module}-calcium/main/mod.ca`
2. `github.com/{author}/{module}/main/mod.ca`
3. `github.com/{author}/{module}-calcium/master/mod.ca`
4. `github.com/{author}/{module}/master/mod.ca`

---

## Boneyard Registry

[Boneyard](https://github.com/ytnobody/boneyard) is the official module registry.

### Registry Structure

```
boneyard/
├── index/
│   └── A/AU/AUTHOR/module-name/
│       ├── meta.toml        # Module metadata
│       ├── 1.0.0.toml       # Version info
│       ├── 1.1.0.toml
│       └── latest.toml      # Latest version
└── tags/
    └── keyword/
        └── A/AU/AUTHOR/
            └── module-name -> (symlink)
```

### meta.toml (in registry)

```toml
name = "module-name"
author = "author"
description = "Description"
license = "MIT"
keywords = ["utility", "helper"]
entry = "mod.ca"
source_url = "https://github.com/author/module-name"
```

### Version TOML

```toml
version = "1.0.0"
tag = "v1.0.0"
commit = "abc123def456..."
published = "2025-01-24T10:00:00Z"
```

### Publishing a Module

1. Create `meta.toml` in your repository:

```toml
name = "my-module"
author = "YOURNAME"
description = "My module"
license = "MIT"
keywords = ["utility"]
entry = "mod.ca"
```

2. (Optional) Create a release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

3. Submit by opening an issue at [Boneyard](https://github.com/ytnobody/boneyard/issues/new):

```
https://raw.githubusercontent.com/YOURNAME/my-module/main/meta.toml
```

---

## Version Constraints

Supported version constraint syntax:

| Constraint | Meaning |
|------------|---------|
| `^1.0.0` | Compatible with 1.x.x (>=1.0.0 <2.0.0) |
| `~1.0.0` | Patch updates only (>=1.0.0 <1.1.0) |
| `>=1.0.0` | Greater than or equal |
| `<=1.0.0` | Less than or equal |
| `>1.0.0` | Greater than |
| `<1.0.0` | Less than |
| `=1.0.0` | Exact version |
| `*` | Any version |

---

## Example Usage

### Complete Workflow

```bash
# Create project
bone init my-app
cd my-app

# Add dependencies
bone add ytnobody/json

# Edit main.ca
cat > main.ca << 'EOF'
use core.io!;
use ytnobody/json;

data = {name: "Calcium", version: 1};
json_str = json.stringify(data);
io.println(json_str);

result = json.parse('{"hello": "world"}');
parsed = result !? { success(v) => v  failure(e) => {} };
io.println(parsed["hello"]);
EOF

# Run
calcium main.ca
```

### Output

```
{"name":"Calcium","version":1}
world
```

---

## Future Enhancements

- [ ] Version range resolution for dependencies
- [ ] Dependency tree visualization
- [ ] Private registry support
- [ ] Workspace/monorepo support
- [ ] `pub use ... as` syntax for re-exports
- [ ] Import maps for URL aliases
