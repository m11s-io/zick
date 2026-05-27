package app

import "github.com/spf13/cobra"

// NewRootCmd builds the full zick command tree. version is injected at build
// time by the main package via ldflags; pass an empty string for tooling use.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "zick",
		Short: "Developer-first supply-chain and secret scanning CLI",
		Long:  "zick checks dependency freshness, scans for secrets, and runs vulnerability scanners locally or through Docker fallback.",
		Example: `  # freshness age gate for the current directory
  zick fresh .

  # full audit: freshness + secrets + vulnerability scan
  zick audit .

  # install a managed pre-commit hook
  zick hook install .`,
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddGroup(
		&cobra.Group{ID: "scan", Title: "Scanning:"},
		&cobra.Group{ID: "workflow", Title: "Workflow:"},
	)

	root.AddCommand(
		newAuditCmd(),
		newFreshCmd(),
		newHookCmd(),
		newScanCmd(),
		newSBOMCmd(),
		newSecretsCmd(),
	)

	return root
}
