package app

import (
	"fmt"

	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/tools"
	"github.com/spf13/cobra"
)

func newSecretsCmd() *cobra.Command {
	var tool string

	cmd := &cobra.Command{
		Use:     "secrets [path]",
		Short:   "Scan for leaked secrets (betterleaks / gitleaks)",
		GroupID: "scan",
		Long: `Runs a secret scanner against the target path. Resolves tool execution in
order: local binary → Docker fallback.

Supported tools: betterleaks, gitleaks, auto (default: tries betterleaks first)`,
		Example: `  # auto-select scanner
  zick secrets .

  # force gitleaks
  zick secrets --tool gitleaks .

  # force betterleaks
  zick secrets --tool betterleaks .`,
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
			if !flagChanged(cmd, "tool") && cfg.Secrets.Tool != "" {
				tool = cfg.Secrets.Tool
			}
			if err := validateSecretsFlags(tool); err != nil {
				return err
			}

			executor := tools.NewExecutor(cmd.OutOrStdout(), cmd.ErrOrStderr())
			return executor.RunSecrets(path, tool)
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "auto", "Secret scanner to use (betterleaks, gitleaks, auto)")

	return cmd
}

func validateSecretsFlags(tool string) error {
	switch tool {
	case "auto", "betterleaks", "gitleaks":
		return nil
	default:
		return fmt.Errorf("--tool must be one of: auto, betterleaks, gitleaks")
	}
}
