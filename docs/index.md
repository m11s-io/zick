# zick

Developer-first supply-chain security CLI.

zick currently provides dependency publish-age checks for npm-compatible
projects, secret scanning through betterleaks or gitleaks, and vulnerability
scanning through osv-scanner or trivy. It can also generate SBOMs with syft.

## Quick Start

```bash
zick fresh .
zick secrets .
zick secrets --tool gitleaks .
zick scan --tools osv-scanner .
zick sbom --output sbom.json .
zick audit .
```

## Commands

| Command | What it does | Stage |
|---------|-------------|-------|
| `zick fresh [path]` | Freshness age gate for npm-compatible dependencies | 1 |
| `zick secrets [path]` | Secret scan via betterleaks or gitleaks | 1 |
| `zick scan [path]` | Vulnerability scan via osv-scanner and trivy | 1 |
| `zick sbom [path]` | SBOM generation via syft | 1 |
| `zick audit [path]` | Full audit: fresh + scan + secrets | 1 |
| `zick hook install` | Install pre-commit hooks | planned |
| `zick serve` | Run as a REST API service | planned |

## Supply Chain Freshness

`zick fresh` queries npm registry metadata for publish timestamps and flags
dependencies published within a configurable age window. The default age gate is
7 days.

Manifest resolution order:

1. `bun.lock` - exact resolved versions
2. `pnpm-lock.yaml` - exact resolved versions
3. `yarn.lock` - exact resolved versions
4. `package-lock.json` - exact installed versions
5. `package.json` - current `latest` version from registry

Flags:

```text
--age-gate int     Flag packages published within this many days (default 7)
--fail-on string   Exit 1 when this risk level is found: high | warn
--format string    Output format: table | json
--include-dev      Include devDependencies for package.json/package-lock.json
```

Ecosystems: npm-compatible registry metadata in Stage 1. PyPI, crates.io,
RubyGems, and Go modules are planned.

## Secret Scanning

`zick secrets` runs a secret scanner against the target path. Tool resolution
order is local binary first, then Docker fallback.

```bash
zick secrets .
zick secrets --tool gitleaks .
```

Supported tools:

- `betterleaks`
- `gitleaks`
- `auto` (currently resolves to betterleaks)

## Vulnerability Scanning

`zick scan` runs vulnerability scanners against the target path. Tool resolution
order is local binary first, then Docker fallback.

```bash
zick scan .
zick scan --tools osv-scanner,trivy .
zick scan --sarif-output zick.sarif .
```

Supported scanners:

- `osv-scanner`
- `trivy`

## SBOM Generation

`zick sbom` generates a software bill of materials with syft.

```bash
zick sbom .
zick sbom --format spdx-json --output sbom.json .
```

Supported formats:

- `cyclonedx-json`
- `spdx-json`
- `syft-json`

## Audit

`zick audit` runs `fresh`, `secrets`, and `scan` in one command.

```bash
zick audit .
zick audit --skip-secrets --scan-tools osv-scanner .
```

## Configuration

Place `.zick.yaml` at your project root. All fields are optional.

```yaml
fresh:
  age_gate_days: 7
  include_dev: false
  fail_on: high
  format: table

secrets:
  tool: auto

scan:
  tools: [osv-scanner, trivy]
  sarif_output: ""

sbom:
  format: cyclonedx-json
  output: ""
```

Config discovery walks upward from the target path until it finds `.zick.yaml`.
Command-line flags override `.zick.yaml`.

## Architecture

```text
cmd/zick/
  main.go         root command + ExecuteContext + SilentError handling
  audit.go        zick audit command
  fresh.go        zick fresh command
  scan.go         zick scan command
  sbom.go         zick sbom command
  secrets.go      zick secrets command

internal/
  cli/
    error.go      SilentError for scan-result exits
  config/
    config.go     .zick.yaml loader
  fresh/
    npm.go        npm registry client + bun.lock / package-lock.json / package.json parser
    resolver.go   age gate classification
  tools/
    executor.go   Tool interface + local -> Docker fallback resolver
    betterleaks.go  betterleaks Tool implementation
    gitleaks.go     gitleaks Tool implementation
    osvscanner.go   osv-scanner Tool implementation
    syft.go         syft Tool implementation
    trivy.go        trivy Tool implementation
```

## Roadmap

Next useful work:

- PyPI, crates.io, RubyGems, and Go module freshness checks
- pre-commit hook installer
