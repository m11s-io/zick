# zick

> Developer-first security scanning CLI — one command, the whole picture.

zick orchestrates best-in-class open-source security tools into a single binary.
Run it locally, in Docker, or point it at a deployed cluster service. No vendor
lock-in, no agents, no accounts required.

---

## Why zick

Modern supply chain attacks exploit a simple window: the gap between when a
package is published and when the community notices something is wrong. Existing
tools catch known CVEs. zick catches the unknown — packages too new to have been
vetted, secrets committed before the push, SBOMs never generated.

It also solves the toolchain fragmentation problem: trivy, osv-scanner,
betterleaks, syft, renovate — all excellent individually, all requiring separate
invocations, configs, and output formats. zick wraps them into a coherent
developer workflow.

---

## Commands

```
zick fresh      Check dependencies for supply chain risk (freshness age gate)
zick scan       Run vulnerability scan (trivy + osv-scanner)
zick secrets    Scan for leaked secrets (betterleaks / gitleaks)
zick sbom       Generate SBOM (syft → CycloneDX or SPDX)
zick audit      Full audit: fresh + scan + secrets in one pass
zick hook       Install / remove pre-commit hooks
zick serve      Run as a local or cluster-deployed API service
```

---

## Supply chain freshness (`zick fresh`)

The flagship feature. Queries package registries for publish timestamps and flags
dependencies published within a configurable age window (default: 7 days).

Supported ecosystems:
- npm / jsr
- PyPI
- crates.io
- RubyGems
- Go modules (proxy.golang.org)

Example output:

```
$ zick fresh package.json

  RISK   PACKAGE              VERSION   PUBLISHED     AGE
  HIGH   some-util            2.1.0     2026-05-25    2 days ago   ← below 7d gate
  WARN   another-pkg          1.0.0     2026-05-21    6 days ago   ← below 7d gate
  OK     lodash               4.17.21   2021-02-20    5 years ago

  2 packages below the 7-day age gate. Review before installing.
```

Complements Renovate's `minimumReleaseAge` for CI — zick gives the same gate
interactively to developers at install time.

---

## Integrated tools

| Tool | Purpose | zick command |
|------|---------|--------------|
| [osv-scanner](https://github.com/google/osv-scanner) | Known CVE matching via OSV database | `zick scan` |
| [trivy](https://github.com/aquasecurity/trivy) | Container + filesystem vulnerability scan | `zick scan` |
| [betterleaks](https://github.com/smartbugs/betterleaks) | Secret detection in code | `zick secrets` |
| [syft](https://github.com/anchore/syft) | SBOM generation (CycloneDX / SPDX) | `zick sbom` |
| [renovate](https://github.com/renovatebot/renovate) | Dependency update policy enforcement | `zick audit` |
| [gitleaks](https://github.com/gitleaks/gitleaks) | Git history secret scan | `zick secrets` |

---

## Execution modes

zick resolves tool execution in order — no config required for basic use:

1. **Local** — uses the installed tool if found in `$PATH`
2. **Docker** — falls back to `docker run ghcr.io/m11s-io/zick-tools:<tool>`
3. **Remote** — forwards to a cluster-deployed `zick serve` instance (`ZICK_SERVER=https://...`)

---

## Configuration

`.zick.yaml` at the project root:

```yaml
fresh:
  age_gate_days: 7         # flag packages newer than this
  ecosystems: [npm, pypi]  # limit scope
  fail_on: high            # exit 1 on HIGH risk findings

scan:
  tools: [trivy, osv-scanner]
  severity: MEDIUM

secrets:
  tools: [betterleaks, gitleaks]

sbom:
  format: cyclonedx-json
  output: sbom.json
```

---

## GitHub Actions integration

```yaml
- uses: m11s-io/zick-action@v1
  with:
    commands: fresh,secrets
    age_gate_days: 7
    fail_on: high
```

---

## Installation

### Local install

```bash
# macOS / Linux (Homebrew)
brew install m11s-io/tap/zick

# Script
curl -sSL https://raw.githubusercontent.com/m11s-io/zick/main/install.sh | sh

# Go install
go install github.com/m11s-io/zick/cmd/zick@latest
```

### Docker

```bash
docker run --rm -v $(pwd):/src ghcr.io/m11s-io/zick fresh /src
```

### Kubernetes (cluster service)

```bash
helm repo add m11s-io https://charts.m11s-io.github.io
helm install zick m11s-io/zick
```

---

## Roadmap

### Stage 1 — CLI foundation + freshness *(current)*

- [x] Project scaffold (Go + Cobra)
- [ ] `zick fresh` — npm freshness check against registry API
- [ ] `zick secrets` — betterleaks integration (local + Docker fallback)
- [ ] GitHub Actions workflow (`zick-action`)
- [ ] Single binary releases via GoReleaser (linux/darwin, amd64/arm64)
- [ ] Docker image published to `ghcr.io/m11s-io/zick`

### Stage 2 — Vulnerability scanning

- [ ] trivy integration (local + Docker fallback)
- [ ] osv-scanner integration
- [ ] `zick scan` unified command
- [ ] Multi-ecosystem freshness (PyPI, crates.io, RubyGems, Go)
- [ ] SARIF output for GitHub Security tab

### Stage 3 — SBOM + audit

- [ ] syft integration (`zick sbom`)
- [ ] `zick audit` combining all checks
- [ ] Pre-commit hook installer (`zick hook`)
- [ ] Renovate config audit helper
- [ ] bun.lock / yarn.lock / pnpm-lock.yaml lockfile parsing

### Stage 4 — Platform

- [ ] `zick serve` REST API
- [ ] Helm chart for Kubernetes deployment
- [ ] Result persistence + history
- [ ] Web dashboard
- [ ] Slack / webhook notifications

---

## Contributing

zick is built on Go + Cobra. Each tool integration lives in its own package
under `internal/tools/`. Adding a new tool means implementing the `Tool`
interface — see `docs/adding-a-tool.md`.

```
cmd/
  zick/
    main.go         ← root command + execute
    fresh.go        ← zick fresh command
    secrets.go      ← zick secrets command
internal/
  fresh/
    npm.go          ← npm registry client + lockfile parsing
    resolver.go     ← age gate classification
  tools/
    executor.go     ← local → docker fallback resolution
    betterleaks.go  ← betterleaks integration
```

---

## License

Apache 2.0
