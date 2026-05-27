# zick

Developer-first security scanning CLI — one command, the whole picture.

zick orchestrates best-in-class open-source security tools into a single binary. Run it locally, in Docker, or point it at a deployed cluster service. No vendor lock-in, no agents, no accounts required.

---

## Why zick

Modern supply chain attacks exploit a simple window: the gap between when a package is published and when the community notices something is wrong. Existing tools catch known CVEs. zick catches the unknown — packages too new to have been vetted, secrets committed before the push, SBOMs never generated.

It also solves fragmentation: trivy, osv-scanner, betterleaks, syft — all excellent individually, all requiring separate invocations, configs, and output formats. zick wraps them into a coherent developer workflow.

---

## Quick start

```bash
# Install (Linux / macOS)
curl -sSL https://raw.githubusercontent.com/m11s-io/zick/main/install.sh | sh

# Or via Go
go install github.com/m11s-io/zick/cmd/zick@latest

# Check your npm dependencies aren't suspiciously fresh
zick fresh .

# Scan for committed secrets
zick secrets .
```

---

## Commands

| Command | What it does | Stage |
|---------|-------------|-------|
| `zick fresh [path]` | Freshness age gate — flags packages published within N days | 1 ✅ |
| `zick secrets [path]` | Secret scan via betterleaks / gitleaks | 1 ✅ |
| `zick scan [path]` | Vulnerability scan via trivy + osv-scanner | 2 |
| `zick sbom [path]` | SBOM generation via syft | 3 |
| `zick audit [path]` | Full audit: fresh + scan + secrets | 3 |
| `zick hook install` | Install pre-commit hooks | 3 |
| `zick serve` | Run as a REST API service | 4 |

---

## Supply chain freshness

The flagship feature. `zick fresh` queries package registries for publish timestamps and flags dependencies published within a configurable age window (default: 7 days).

```
$ zick fresh .

  RISK   PACKAGE              VERSION   PUBLISHED     AGE
  HIGH   some-util            2.1.0     2026-05-25    2 days ago
  WARN   another-pkg          1.0.0     2026-05-21    6 days ago
  OK     lodash               4.17.21   2021-02-20    5 years ago

  2 package(s) below the 7-day age gate.
```

Exit code 1 when violations are found — safe to use in CI.

**Manifest resolution order:**

1. `package-lock.json` → exact installed versions
2. `package.json` → latest matching version from registry

**Flags:**

```
--age-gate int     Flag packages published within this many days (default 7)
--fail-on string   Exit 1 when this risk level is found: high | warn (default "high")
--include-dev      Include devDependencies
```

**Ecosystems:** npm (Stage 1) · PyPI · crates.io · RubyGems · Go modules (Stage 2)

---

## Secret scanning

`zick secrets` runs a secret scanner against the target path. Tool resolution order: local binary → Docker fallback.

```bash
zick secrets .
zick secrets --tool gitleaks .
```

**Supported tools:** betterleaks (default), gitleaks (Stage 2)

---

## Execution modes

zick resolves where to run a tool — no config required for basic use:

| Priority | Mode | Requirement |
|----------|------|-------------|
| 1 | Local | tool binary in `$PATH` |
| 2 | Docker | `docker` in `$PATH` |
| 3 | Remote | `ZICK_SERVER=https://...` env var (Stage 4) |

---

## Configuration

Place `.zick.yaml` at your project root. All fields are optional.

```yaml
fresh:
  age_gate_days: 7      # default: 7
  include_dev: false
  fail_on: high         # high | warn

secrets:
  tool: auto            # betterleaks | gitleaks | auto
```

---

## GitHub Actions

```yaml
- uses: m11s-io/zick-action@v1
  with:
    commands: fresh,secrets
    age_gate_days: 7
    fail_on: high
```

---

## Architecture

```
cmd/zick/
  main.go         newRootCmd() + ExecuteContext + SilentError handling
  fresh.go        zick fresh — uses cmd.OutOrStdout() / cmd.ErrOrStderr()
  secrets.go      zick secrets — passes cobra IO writers to executor

internal/
  cli/
    error.go      SilentError — exit code without printing a message
  fresh/
    npm.go        npm registry client + package-lock.json / package.json parser
    resolver.go   age gate classification (RiskOK / RiskWarn / RiskHigh)
  tools/
    executor.go   Tool interface + local → Docker fallback resolver
    betterleaks.go  betterleaks Tool implementation
```

**Key design decisions:**

- `newRootCmd()` returns a fresh command tree — enables testing without global state
- All IO flows through `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` — the executor receives `io.Writer` pairs, so tests can capture output
- `SilentError` separates "scan found violations (exit 1, already printed summary)" from "tool failed (print error + exit 1)"
- `ExecuteContext` threads a `context.Context` through cobra — subcommands can call `cmd.Context()` for timeouts and config values as the project grows
- Command groups (`AddGroup`) are wired now; Stage 2 commands get `GroupID: "scan"` and slot into help output without a refactor

---

## Contributing

Each tool integration is a struct implementing the `Tool` interface in `internal/tools/`:

```go
type Tool interface {
    Name() string
    BinaryName() string
    DockerImage() string
    Args(path string) []string
}
```

Add the struct, register it in `executor.go`'s `RunSecrets` (or a future `RunScan`), done.
