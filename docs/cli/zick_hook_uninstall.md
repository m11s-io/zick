---
title: "zick hook uninstall"
slug: "zick_hook_uninstall"
description: "CLI reference for zick hook uninstall"
---

## zick hook uninstall

Remove zick pre-commit hook

### Synopsis

Removes the zick-managed pre-commit hook. Fails if the hook is not managed by zick unless --force is passed.

```
zick hook uninstall [path] [flags]
```

### Examples

```
  zick hook uninstall .
  zick hook uninstall --force .
```

### Options

```
      --force   Remove an existing unmanaged pre-commit hook
  -h, --help    help for uninstall
```

### SEE ALSO

* [zick hook](zick_hook.md)	 - Install or remove Git hooks

