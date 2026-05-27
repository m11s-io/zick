# zick

> Developer-first supply-chain and secret scanning CLI.

zick currently provides three local developer checks:

- dependency publish-age checks for npm-compatible projects
- secret scanning through betterleaks or gitleaks
- vulnerability scanning through osv-scanner or trivy

## Why zick

Modern supply chain attacks exploit the gap between when a package is published
and when the community notices something is wrong. `zick fresh` helps make that
gap visible by flagging packages that are newer than your configured age gate.

`zick secrets` gives the same project-level entry point for local secret
scanners, using an installed tool when available and Docker as a fallback.

## Commands

```text
zick fresh      Check dependencies for supply chain risk (freshness age gate)
zick secrets    Scan for leaked secrets (betterleaks / gitleaks)
zick scan       Run vulnerability scan (osv-scanner / trivy)
```

## Supply Chain Freshness

`zick fresh` queries npm registry metadata for publish timestamps and flags
dependencies published within a configurable age window. The default age gate is
7 days.

Supported inputs:

- `bun.lock`
- `package-lock.json`
- `package.json`

With a lockfile, zick checks exact resolved versions. With only `package.json`,
zick checks the current registry `latest` version for each dependency.

```bash
zick fresh .
zick fresh --age-gate 14 --fail-on warn --include-dev .
zick fresh --format json .
```

```text
RISK   PACKAGE      VERSION   PUBLISHED    AGE
HIGH   some-util    2.1.0     2026-05-25   2 days ago
WARN   another-pkg  1.0.0     2026-05-21   6 days ago
OK     lodash       4.17.21   2021-02-20   5 years ago

2 package(s) below the 7-day age gate.
```

Flags:

```text
--age-gate int     Flag packages published within this many days (default 7)
--fail-on string   Exit 1 when this risk level is found: high | warn (default "high")
--format string    Output format: table | json (default "table")
--include-dev      Include devDependencies for package.json/package-lock.json
```

## Secret Scanning

`zick secrets` runs a secret scanner against the target path.

```bash
zick secrets .
zick secrets --tool gitleaks .
```

Supported tools:

- `betterleaks`
- `gitleaks`
- `auto` (currently resolves to betterleaks)

For external tools, zick resolves execution in order:

1. Local binary in `$PATH`
2. Docker fallback using the tool's container image

## Vulnerability Scanning

`zick scan` runs vulnerability scanners against the target path.

```bash
zick scan .
zick scan --tools osv-scanner .
zick scan --tools osv-scanner,trivy .
```

Supported scanners:

- `osv-scanner`
- `trivy`

Like secret scanning, zick uses a local binary first and falls back to Docker.

## Configuration

Place `.zick.yaml` at the project root. All fields are optional.

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
```

Command-line flags override `.zick.yaml`.

## GitHub Actions

```yaml
- uses: m11s-io/zick-action@v1
  with:
    commands: fresh,secrets,scan
    age_gate_days: 7
    fail_on: high
    secrets_tool: auto
    scan_tools: osv-scanner,trivy
```

## Installation

```bash
# macOS / Linux (Homebrew)
brew install m11s-io/tap/zick

# Script
curl -sSL https://raw.githubusercontent.com/m11s-io/zick/main/install.sh | sh

# Go install
go install github.com/m11s-io/zick/cmd/zick@latest
```

Docker:

```bash
docker run --rm -v "$(pwd):/src" ghcr.io/m11s-io/zick fresh /src
```

## Roadmap

Stage 1 - CLI foundation and freshness:

- [x] Project scaffold (Go + Cobra)
- [x] `zick fresh` npm registry freshness check
- [x] `bun.lock`, `package-lock.json`, and `package.json` parsing
- [x] `.zick.yaml` for `fresh` and `secrets`
- [x] `zick secrets` with betterleaks and gitleaks
- [x] `zick scan` with osv-scanner and trivy
- [x] GitHub Actions workflow (`zick-action`)
- [x] GitHub Action local smoke workflow
- [x] Single binary release configuration via GoReleaser
- [x] Docker image release configuration for `ghcr.io/m11s-io/zick`

Stage 2 - Vulnerability scanning:

- [ ] Multi-ecosystem freshness: PyPI, crates.io, RubyGems, Go
- [ ] SARIF output for GitHub Security tab

Stage 3 - SBOM and audit:

- [ ] syft integration (`zick sbom`)
- [ ] `zick audit` combining all checks
- [ ] Pre-commit hook installer (`zick hook`)
- [ ] Renovate config audit helper
- [ ] yarn.lock / pnpm-lock.yaml parsing

Stage 4 - Platform:

- [ ] `zick serve` REST API
- [ ] Helm chart for Kubernetes deployment
- [ ] Result persistence and history
- [ ] Web dashboard
- [ ] Slack / webhook notifications

## Contributing

zick is built on Go + Cobra.

```text
cmd/
  zick/
    main.go         root command + execute
    fresh.go        zick fresh command
    scan.go         zick scan command
    secrets.go      zick secrets command
internal/
  config/
    config.go       .zick.yaml loader
  fresh/
    npm.go          npm registry client + lockfile parsing
    resolver.go     age gate classification
  tools/
    executor.go     local -> Docker fallback resolution
    betterleaks.go  betterleaks integration
    gitleaks.go     gitleaks integration
    osvscanner.go   osv-scanner integration
    trivy.go        trivy integration
```

## License

Apache 2.0
