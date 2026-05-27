---
title: "zick hook install"
slug: "zick_hook_install"
description: "CLI reference for zick hook install"
---

## zick hook install

Install zick pre-commit hook

### Synopsis

Installs a managed pre-commit hook in the target Git repository.
By default the hook runs zick fresh; add --secrets to also run secret scanning.

An existing unmanaged hook is left untouched unless --force is passed.

```
zick hook install [path] [flags]
```

### Examples

```
  # freshness-only hook (default)
  zick hook install .

  # include secret scanning with gitleaks
  zick hook install --secrets --secrets-tool gitleaks .

  # replace an unmanaged hook
  zick hook install --force .
```

### Options

```
      --force                 Replace an existing unmanaged pre-commit hook
  -h, --help                  help for install
      --secrets               Run zick secrets from the pre-commit hook
      --secrets-tool string   Secret scanner to use when --secrets is enabled (default "auto")
```

### SEE ALSO

* [zick hook](zick_hook.md)	 - Install or remove Git hooks

