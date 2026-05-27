---
title: "zick scan"
slug: "zick_scan"
description: "CLI reference for zick scan"
---

## zick scan

Run vulnerability scan (osv-scanner / trivy)

### Synopsis

Runs vulnerability scanners against the target path. Resolves tool execution
in order: local binary → Docker fallback.

Supported tools: osv-scanner, trivy

```
zick scan [path] [flags]
```

### Examples

```
  # run both default scanners
  zick scan .

  # osv-scanner only, write SARIF output
  zick scan --tools osv-scanner --sarif-output results.sarif .

  # trivy only
  zick scan --tools trivy .
```

### Options

```
  -h, --help                  help for scan
      --sarif-output string   Write scanner output as SARIF to this path
      --tools string          Comma-separated scanners to run (osv-scanner, trivy) (default "osv-scanner,trivy")
```

### SEE ALSO

* [zick](zick.md)	 - Developer-first supply-chain and secret scanning CLI

