package app

import (
	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/hook"
	"github.com/spf13/cobra"
)

func newHookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hook",
		Short:   "Install or remove Git hooks",
		GroupID: "workflow",
		Example: `  zick hook install .
  zick hook install --secrets --secrets-tool gitleaks .
  zick hook uninstall .`,
	}

	cmd.AddCommand(newHookInstallCmd(), newHookUninstallCmd())
	return cmd
}

func newHookInstallCmd() *cobra.Command {
	var includeSecrets bool
	var secretsTool string
	var force bool

	cmd := &cobra.Command{
		Use:   "install [path]",
		Short: "Install zick pre-commit hook",
		Long: `Installs a managed pre-commit hook in the target Git repository.
By default the hook runs zick fresh; add --secrets to also run secret scanning.

An existing unmanaged hook is left untouched unless --force is passed.`,
		Example: `  # freshness-only hook (default)
  zick hook install .

  # include secret scanning with gitleaks
  zick hook install --secrets --secrets-tool gitleaks .

  # replace an unmanaged hook
  zick hook install --force .`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if !flagChanged(cmd, "secrets") && cfg.Hook.IncludeSecrets != nil {
				includeSecrets = *cfg.Hook.IncludeSecrets
			}
			if !flagChanged(cmd, "secrets-tool") && cfg.Hook.SecretsTool != "" {
				secretsTool = cfg.Hook.SecretsTool
			}
			if secretsTool == "" {
				secretsTool = "auto"
			}
			if err := validateSecretsFlags(secretsTool); err != nil {
				return err
			}

			hookPath, err := hook.Install(path, hook.InstallOptions{
				IncludeSecrets: includeSecrets,
				SecretsTool:    secretsTool,
				Force:          force,
			})
			if err != nil {
				return err
			}
			cmd.Printf("Installed pre-commit hook: %s\n", hookPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&includeSecrets, "secrets", false, "Run zick secrets from the pre-commit hook")
	cmd.Flags().StringVar(&secretsTool, "secrets-tool", "auto", "Secret scanner to use when --secrets is enabled")
	cmd.Flags().BoolVar(&force, "force", false, "Replace an existing unmanaged pre-commit hook")
	return cmd
}

func newHookUninstallCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall [path]",
		Short: "Remove zick pre-commit hook",
		Long:  "Removes the zick-managed pre-commit hook. Fails if the hook is not managed by zick unless --force is passed.",
		Example: `  zick hook uninstall .
  zick hook uninstall --force .`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			hookPath, err := hook.Uninstall(path, force)
			if err != nil {
				return err
			}
			cmd.Printf("Removed pre-commit hook: %s\n", hookPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Remove an existing unmanaged pre-commit hook")
	return cmd
}
