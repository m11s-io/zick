---
title: "zick secrets"
slug: "zick_secrets"
description: "CLI reference for zick secrets"
---

## zick secrets

Scan for leaked secrets (betterleaks / gitleaks)

### Synopsis

Runs a secret scanner against the target path. Resolves tool execution in
order: local binary → Docker fallback.

Supported tools: betterleaks, gitleaks, auto (default: tries betterleaks first)

```
zick secrets [path] [flags]
```

### Examples

```
  # auto-select scanner
  zick secrets .

  # force gitleaks
  zick secrets --tool gitleaks .

  # force betterleaks
  zick secrets --tool betterleaks .
```

### Options

```
  -h, --help          help for secrets
      --tool string   Secret scanner to use (betterleaks, gitleaks, auto) (default "auto")
```

### SEE ALSO

* [zick](zick.md)	 - Developer-first supply-chain and secret scanning CLI

