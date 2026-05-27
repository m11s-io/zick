package app

import (
	"fmt"

	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/tools"
	"github.com/spf13/cobra"
)

func newSBOMCmd() *cobra.Command {
	var format string
	var output string

	cmd := &cobra.Command{
		Use:     "sbom [path]",
		Short:   "Generate SBOM (syft)",
		GroupID: "scan",
		Long: `Generates a software bill of materials using syft. Resolves execution in
order: local binary → Docker fallback.`,
		Example: `  # CycloneDX JSON to stdout (default)
  zick sbom .

  # SPDX JSON written to a file
  zick sbom --format spdx-json --output sbom.json .

  # syft native format
  zick sbom --format syft-json --output sbom.syft.json .`,
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
			if !flagChanged(cmd, "format") && cfg.SBOM.Format != "" {
				format = cfg.SBOM.Format
			}
			if !flagChanged(cmd, "output") && cfg.SBOM.Output != "" {
				output = cfg.SBOM.Output
			}
			if err := validateSBOMFlags(format); err != nil {
				return err
			}

			executor := tools.NewExecutor(cmd.OutOrStdout(), cmd.ErrOrStderr())
			return executor.RunSBOM(path, tools.SBOMOptions{Format: format, Output: output})
		},
	}

	cmd.Flags().StringVar(&format, "format", "cyclonedx-json", "SBOM format (cyclonedx-json, spdx-json, syft-json)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Write SBOM to this file")

	return cmd
}

func validateSBOMFlags(format string) error {
	switch format {
	case "cyclonedx-json", "spdx-json", "syft-json":
		return nil
	default:
		return fmt.Errorf("--format must be one of: cyclonedx-json, spdx-json, syft-json")
	}
}
