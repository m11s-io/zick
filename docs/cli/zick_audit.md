---
title: "zick audit"
slug: "zick_audit"
description: "CLI reference for zick audit"
---

## zick audit

Run fresh, secrets, and scan checks

### Synopsis

Runs all three checks in sequence: dependency freshness, secret scanning,
and vulnerability scanning. Produces optional JSON and HTML reports.

```
zick audit [path] [flags]
```

### Examples

```
  # full audit of the current directory
  zick audit .

  # skip secrets, write machine-readable and HTML reports
  zick audit --skip-secrets --json-output report.json --html-output report.html .

  # only freshness + secrets, strict age gate
  zick audit --skip-scan --age-gate 3 .
```

### Options

```
      --age-gate int          Flag packages published within this many days (default 7)
      --fail-on string        Exit 1 when this risk level is found (high, warn) (default "high")
  -h, --help                  help for audit
      --html-output string    Write self-contained audit report HTML to this path
      --include-dev           Include devDependencies
      --json-output string    Write audit report JSON to this path
      --sarif-output string   Write scanner output as SARIF to this path
      --scan-tools string     Comma-separated scanners to run (default "osv-scanner,trivy")
      --secrets-tool string   Secret scanner to use (auto, betterleaks, gitleaks) (default "auto")
      --skip-fresh            Skip dependency freshness check
      --skip-scan             Skip vulnerability scanning
      --skip-secrets          Skip secret scanning
```

### SEE ALSO

* [zick](zick.md)	 - Developer-first supply-chain and secret scanning CLI

