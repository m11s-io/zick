---
title: "zick fresh"
slug: "zick_fresh"
description: "CLI reference for zick fresh"
---

## zick fresh

Check dependencies for supply chain risk (freshness age gate)

### Synopsis

Queries package registries for publish timestamps and flags dependencies
published within the configured age gate. Helps catch supply chain attacks
before packages are installed.

Reads bun.lock, pnpm-lock.yaml, yarn.lock, package-lock.json, or package.json.

```
zick fresh [path] [flags]
```

### Examples

```
  # check current directory with the default 7-day gate
  zick fresh .

  # stricter gate, include devDependencies, JSON output
  zick fresh --age-gate 3 --include-dev --format json .

  # exit 1 on any package below the warn threshold
  zick fresh --fail-on warn .
```

### Options

```
      --age-gate int     Flag packages published within this many days (default 7)
      --fail-on string   Exit 1 when this risk level is found (high, warn) (default "high")
      --format string    Output format (table, json) (default "table")
  -h, --help             help for fresh
      --include-dev      Include devDependencies
```

### SEE ALSO

* [zick](zick.md)	 - Developer-first supply-chain and secret scanning CLI

