# Development Scripts

This directory contains helper scripts used during development and CI/CD.

## Active Scripts

### `gofmtcheck.sh`

Checks if Go code needs formatting with `gofmt`.

**Usage:**
```bash
make fmtcheck
# or directly:
./scripts/gofmtcheck.sh
```

**Called by:**
- `make build`
- `make testacc`
- `make fmtcheck`

**Purpose:** Ensures all Go code follows standard formatting conventions before building or testing.

**Exit codes:**
- `0` - All files are properly formatted
- `1` - Some files need formatting (run `make fmt` to fix)

---

### `errcheck.sh`

Checks for unchecked errors in Go code.

**Usage:**
```bash
make errcheck
# or directly:
./scripts/errcheck.sh
```

**Called by:**
- Manual invocation only (optional check)

**Purpose:** Static analysis tool to find places where error return values are not checked. This helps catch potential bugs where errors are silently ignored.

**Note:** This is an optional check and not run automatically by CI/CD. It's useful to run before submitting PRs.

**Exit codes:**
- `0` - No unchecked errors found
- `1` - Unchecked errors detected (review and handle appropriately)

---

## Removed Scripts

The following scripts were previously in this directory but have been removed:

- **`circle-ci.sh`** - Removed (no Circle CI configuration exists; GitHub Actions used instead)
- **`gogetcookie.sh`** - Removed (legacy Google Source authentication; no longer needed)
- **`changelog-links.sh`** - Removed (manual changelog formatting; modern workflow doesn't require this)

---

## GitHub Workflows

Note that the GitHub Actions workflows (`.github/workflows/`) **do not use these scripts**. They run checks inline:

- **test.yml**: Runs `go generate` and diff checks directly
- **release.yml**: Uses GoReleaser for releases

These scripts are primarily for **local development** and **Makefile targets**.

---

## Adding New Scripts

If you add new development scripts:

1. Place them in this directory
2. Make them executable: `chmod +x scripts/your-script.sh`
3. Document them in this README
4. Add corresponding Makefile target if appropriate
5. Consider if they should be integrated into CI/CD workflows

