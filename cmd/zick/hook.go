package main

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
		Args:  cobra.MaximumNArgs(1),
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
		Args:  cobra.MaximumNArgs(1),
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
