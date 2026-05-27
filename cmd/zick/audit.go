package main

import (
	"github.com/m11s-io/zick/internal/cli"
	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/fresh"
	"github.com/m11s-io/zick/internal/tools"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	var ageDays int
	var failOn string
	var includeDev bool
	var scanToolsFlag string
	var secretsTool string
	var skipFresh bool
	var skipSecrets bool
	var skipScan bool
	var sarifOutput string

	cmd := &cobra.Command{
		Use:     "audit [path]",
		Short:   "Run fresh, secrets, and scan checks",
		GroupID: "scan",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			if !flagChanged(cmd, "age-gate") && cfg.Fresh.AgeGateDays != nil {
				ageDays = *cfg.Fresh.AgeGateDays
			}
			if !flagChanged(cmd, "include-dev") && cfg.Fresh.IncludeDev != nil {
				includeDev = *cfg.Fresh.IncludeDev
			}
			if !flagChanged(cmd, "fail-on") && cfg.Fresh.FailOn != "" {
				failOn = cfg.Fresh.FailOn
			}
			if !flagChanged(cmd, "secrets-tool") && cfg.Secrets.Tool != "" {
				secretsTool = cfg.Secrets.Tool
			}
			if !flagChanged(cmd, "sarif-output") && cfg.Scan.SARIFOutput != "" {
				sarifOutput = cfg.Scan.SARIFOutput
			}

			scanTools := splitTools(scanToolsFlag)
			if !flagChanged(cmd, "scan-tools") && len(cfg.Scan.Tools) > 0 {
				scanTools = cfg.Scan.Tools
			}
			if err := validateFreshFlags(ageDays, failOn, "table"); err != nil {
				return err
			}
			if err := validateSecretsFlags(secretsTool); err != nil {
				return err
			}
			if !skipScan {
				if err := validateScanTools(scanTools); err != nil {
					return err
				}
			}

			failed := false
			if !skipFresh {
				cmd.Println("== fresh ==")
				results, err := fresh.Check(path, fresh.Options{
					AgeDays:    ageDays,
					IncludeDev: includeDev,
					ErrOut:     cmd.ErrOrStderr(),
				})
				if err != nil {
					return err
				}
				if len(results) == 0 {
					cmd.Println("No supported manifest found (bun.lock, pnpm-lock.yaml, yarn.lock, package-lock.json, package.json).")
				} else {
					printFreshTable(cmd, results)
					violations := countViolations(results, failOn)
					if violations > 0 {
						cmd.PrintErrf("\n%d package(s) below the %d-day age gate.\n", violations, ageDays)
						failed = true
					}
				}
			}

			executor := tools.NewExecutor(cmd.OutOrStdout(), cmd.ErrOrStderr())
			if !skipSecrets {
				cmd.Println("== secrets ==")
				if err := executor.RunSecrets(path, secretsTool); err != nil {
					return err
				}
			}
			if !skipScan {
				cmd.Println("== scan ==")
				if err := executor.RunScan(path, scanTools, tools.ScanOptions{SARIFOutput: sarifOutput}); err != nil {
					return err
				}
			}

			if failed {
				return &cli.SilentError{Code: 1}
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&ageDays, "age-gate", 7, "Flag packages published within this many days")
	cmd.Flags().StringVar(&failOn, "fail-on", "high", "Exit 1 when this risk level is found (high, warn)")
	cmd.Flags().BoolVar(&includeDev, "include-dev", false, "Include devDependencies")
	cmd.Flags().StringVar(&secretsTool, "secrets-tool", "auto", "Secret scanner to use (auto, betterleaks, gitleaks)")
	cmd.Flags().StringVar(&scanToolsFlag, "scan-tools", "osv-scanner,trivy", "Comma-separated scanners to run")
	cmd.Flags().StringVar(&sarifOutput, "sarif-output", "", "Write scanner output as SARIF to this path")
	cmd.Flags().BoolVar(&skipFresh, "skip-fresh", false, "Skip dependency freshness check")
	cmd.Flags().BoolVar(&skipSecrets, "skip-secrets", false, "Skip secret scanning")
	cmd.Flags().BoolVar(&skipScan, "skip-scan", false, "Skip vulnerability scanning")

	return cmd
}
