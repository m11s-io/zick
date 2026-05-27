package main

import (
	"fmt"
	"strings"

	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/tools"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var toolList string
	var sarifOutput string

	cmd := &cobra.Command{
		Use:     "scan [path]",
		Short:   "Run vulnerability scan (osv-scanner / trivy)",
		GroupID: "scan",
		Long: `Runs vulnerability scanners against the target path. Resolves tool execution
in order: local binary → Docker fallback.

Supported tools: osv-scanner, trivy`,
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

			scanTools := splitTools(toolList)
			if !flagChanged(cmd, "tools") && len(cfg.Scan.Tools) > 0 {
				scanTools = cfg.Scan.Tools
			}
			if !flagChanged(cmd, "sarif-output") && cfg.Scan.SARIFOutput != "" {
				sarifOutput = cfg.Scan.SARIFOutput
			}
			if err := validateScanTools(scanTools); err != nil {
				return err
			}

			executor := tools.NewExecutor(cmd.OutOrStdout(), cmd.ErrOrStderr())
			return executor.RunScan(path, scanTools, tools.ScanOptions{SARIFOutput: sarifOutput})
		},
	}

	cmd.Flags().StringVar(&toolList, "tools", "osv-scanner,trivy", "Comma-separated scanners to run (osv-scanner, trivy)")
	cmd.Flags().StringVar(&sarifOutput, "sarif-output", "", "Write scanner output as SARIF to this path")

	return cmd
}

func splitTools(value string) []string {
	parts := strings.Split(value, ",")
	tools := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tools = append(tools, part)
		}
	}
	return tools
}

func validateScanTools(scanTools []string) error {
	if len(scanTools) == 0 {
		return fmt.Errorf("--tools must include at least one of: osv-scanner, trivy")
	}
	for _, tool := range scanTools {
		switch tool {
		case "osv-scanner", "trivy":
			continue
		default:
			return fmt.Errorf("--tools contains unsupported scanner %q (supported: osv-scanner, trivy)", tool)
		}
	}
	return nil
}
