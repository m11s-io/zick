---
title: "zick sbom"
slug: "zick_sbom"
description: "CLI reference for zick sbom"
---

## zick sbom

Generate SBOM (syft)

### Synopsis

Generates a software bill of materials using syft. Resolves execution in
order: local binary → Docker fallback.

```
zick sbom [path] [flags]
```

### Examples

```
  # CycloneDX JSON to stdout (default)
  zick sbom .

  # SPDX JSON written to a file
  zick sbom --format spdx-json --output sbom.json .

  # syft native format
  zick sbom --format syft-json --output sbom.syft.json .
```

### Options

```
      --format string   SBOM format (cyclonedx-json, spdx-json, syft-json) (default "cyclonedx-json")
  -h, --help            help for sbom
  -o, --output string   Write SBOM to this file
```

### SEE ALSO

* [zick](zick.md)	 - Developer-first supply-chain and secret scanning CLI

