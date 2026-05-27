package main

import (
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
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			executor := tools.NewExecutor(cmd.OutOrStdout(), cmd.ErrOrStderr())
			return executor.RunSecrets(path, tool)
		},
	}

	cmd.Flags().StringVar(&tool, "tool", "auto", "Secret scanner to use (betterleaks, gitleaks, auto)")

	return cmd
}
