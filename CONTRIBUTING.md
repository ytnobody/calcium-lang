# Contributing to Calcium

Thanks for contributing to Calcium!

## Workflow

1. Make changes on a feature branch.
2. Open a pull request to `main`.
3. CI must pass, including the required **`CI / verification`** status check.
4. A maintainer merges the PR.

## CI and branch protection

`main` requires the **`CI / verification`** check (the `verification` aggregator
job in `.github/workflows/ci.yml`). You cannot merge to `main` unless every job
(`test`, `lint`, `build`) is green.

### Why a single required check

The required check is a single stable name so that changing the matrix, adding a
job, or adjusting steps does not require re-configuring GitHub branch protection
each time.

### If you rename or restructure CI jobs

`CI / verification` = workflow name `CI` + job name `verification`. If you
**rename the workflow or that job**, you MUST also update the branch protection
`required_status_checks.contexts` on GitHub (Settings → Branches → main) to match
the new `<workflow>/<job>` name, or merges to `main` will be blocked.

Changing any *other* job (e.g. adding a `test (1.24)` matrix row) does **not**
require a protection update, because the aggregator still emits the same final
check.

## Calcium source linting

Run the language's own static linter on source files before opening a PR:

```bash
go build -o /tmp/calcium ./cmd/calcium
/tmp/calcium lint <file.ca> ...
```

The CI `lint` job also runs `calcium lint` over the well-formed example files.
If you add a new example that uses constraints or types deliberately, keep any
intentional constraint violations out of the files the CI lint check scans.

## Adding a builtin

If you add a function to the standard library (e.g. `core.string`):

1. Implement it (primitive in Go + wrapper in `pkg/vm/stdlib/core/*.ca`).
2. **Document it in `calcium-spec.md`** — the CI job
   `Check spec covers core.string stdlib` (`scripts/check-spec-stdlib.sh`) fails
   if an implemented function is undocumented.
3. Add a test.

## Releases

Releases are tagged automatically on merge to `main` (patch bump via
`.github/workflows/auto-release.yml`) and published by GoReleaser
(`.goreleaser.yml` with the `Release` workflow). Because tags pushed by the
repository `GITHUB_TOKEN` do not trigger other workflows, `auto-release.yml`
explicitly dispatches the `Release` workflow (`workflow_dispatch`, an allowed
exception) for each new tag. The `Release` workflow can also be run manually
for any tag, and the `Backfill Releases` workflow publishes a Release for every
tag that lacks one. A scheduled `Release Sync Check` fails if the latest tag
ever lacks a Release again. Breaking changes are called out in `CHANGELOG.md`.