---
title: "zick"
slug: "zick"
description: "CLI reference for zick"
---

## zick

Developer-first supply-chain and secret scanning CLI

### Synopsis

zick checks dependency freshness, scans for secrets, and runs vulnerability scanners locally or through Docker fallback.

### Examples

```
  # freshness age gate for the current directory
  zick fresh .

  # full audit: freshness + secrets + vulnerability scan
  zick audit .

  # install a managed pre-commit hook
  zick hook install .
```

### Options

```
  -h, --help   help for zick
```

### SEE ALSO

* [zick audit](zick_audit.md)	 - Run fresh, secrets, and scan checks
* [zick fresh](zick_fresh.md)	 - Check dependencies for supply chain risk (freshness age gate)
* [zick hook](zick_hook.md)	 - Install or remove Git hooks
* [zick sbom](zick_sbom.md)	 - Generate SBOM (syft)
* [zick scan](zick_scan.md)	 - Run vulnerability scan (osv-scanner / trivy)
* [zick secrets](zick_secrets.md)	 - Scan for leaked secrets (betterleaks / gitleaks)

