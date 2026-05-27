package main

import (
	"bytes"
	"io"
	"time"

	"github.com/m11s-io/zick/internal/cli"
	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/fresh"
	"github.com/m11s-io/zick/internal/report"
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
	var jsonOutput string
	var htmlOutput string

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
			if !flagChanged(cmd, "json-output") && cfg.Report.JSONOutput != "" {
				jsonOutput = cfg.Report.JSONOutput
			}
			if !flagChanged(cmd, "html-output") && cfg.Report.HTMLOutput != "" {
				htmlOutput = cfg.Report.HTMLOutput
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

			started := time.Now()
			rep := report.New(path)
			failed := false
			if !skipFresh {
				freshStarted := time.Now()
				cmd.Println("== fresh ==")
				results, err := fresh.Check(path, fresh.Options{
					AgeDays:    ageDays,
					IncludeDev: includeDev,
					ErrOut:     cmd.ErrOrStderr(),
				})
				if err != nil {
					rep.Fresh = report.FreshSection{
						Status:      "failed",
						Duration:    time.Since(freshStarted).Round(time.Millisecond).String(),
						AgeGateDays: ageDays,
						IncludeDev:  includeDev,
						FailOn:      failOn,
						Error:       err.Error(),
					}
					if reportErr := writeAuditReports(cmd, rep, started, jsonOutput, htmlOutput); reportErr != nil {
						return reportErr
					}
					return err
				}
				rep.Fresh = report.FreshSection{
					Status:      "passed",
					Duration:    time.Since(freshStarted).Round(time.Millisecond).String(),
					AgeGateDays: ageDays,
					IncludeDev:  includeDev,
					FailOn:      failOn,
					Results:     report.FreshResults(results),
				}
				if len(results) == 0 {
					rep.Fresh.NoManifest = true
					cmd.Println("No supported manifest found (bun.lock, pnpm-lock.yaml, yarn.lock, package-lock.json, package.json).")
				} else {
					printFreshTable(cmd, results)
					violations := countViolations(results, failOn)
					rep.Fresh.ViolationCnt = violations
					if violations > 0 {
						rep.Fresh.Status = "failed"
						cmd.PrintErrf("\n%d package(s) below the %d-day age gate.\n", violations, ageDays)
						failed = true
					}
				}
			} else {
				rep.Fresh.Status = "skipped"
			}

			if !skipSecrets {
				secretsStarted := time.Now()
				var out bytes.Buffer
				cmd.Println("== secrets ==")
				executor := tools.NewExecutor(io.MultiWriter(cmd.OutOrStdout(), &out), cmd.ErrOrStderr())
				if err := executor.RunSecrets(path, secretsTool); err != nil {
					rep.Secrets = report.ToolSection{
						Status:   "failed",
						Duration: time.Since(secretsStarted).Round(time.Millisecond).String(),
						Tool:     secretsTool,
						Output:   out.String(),
						Error:    err.Error(),
					}
					if reportErr := writeAuditReports(cmd, rep, started, jsonOutput, htmlOutput); reportErr != nil {
						return reportErr
					}
					return err
				}
				rep.Secrets = report.ToolSection{
					Status:   "passed",
					Duration: time.Since(secretsStarted).Round(time.Millisecond).String(),
					Tool:     secretsTool,
					Output:   out.String(),
				}
			} else {
				rep.Secrets.Status = "skipped"
			}
			if !skipScan {
				scanStarted := time.Now()
				var out bytes.Buffer
				cmd.Println("== scan ==")
				executor := tools.NewExecutor(io.MultiWriter(cmd.OutOrStdout(), &out), cmd.ErrOrStderr())
				if err := executor.RunScan(path, scanTools, tools.ScanOptions{SARIFOutput: sarifOutput}); err != nil {
					rep.Scan = report.ToolSection{
						Status:   "failed",
						Duration: time.Since(scanStarted).Round(time.Millisecond).String(),
						Tools:    scanTools,
						Output:   out.String(),
						Error:    err.Error(),
					}
					if reportErr := writeAuditReports(cmd, rep, started, jsonOutput, htmlOutput); reportErr != nil {
						return reportErr
					}
					return err
				}
				rep.Scan = report.ToolSection{
					Status:   "passed",
					Duration: time.Since(scanStarted).Round(time.Millisecond).String(),
					Tools:    scanTools,
					Output:   out.String(),
				}
			} else {
				rep.Scan.Status = "skipped"
			}

			if err := writeAuditReports(cmd, rep, started, jsonOutput, htmlOutput); err != nil {
				return err
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
	cmd.Flags().StringVar(&jsonOutput, "json-output", "", "Write audit report JSON to this path")
	cmd.Flags().StringVar(&htmlOutput, "html-output", "", "Write self-contained audit report HTML to this path")
	cmd.Flags().BoolVar(&skipFresh, "skip-fresh", false, "Skip dependency freshness check")
	cmd.Flags().BoolVar(&skipSecrets, "skip-secrets", false, "Skip secret scanning")
	cmd.Flags().BoolVar(&skipScan, "skip-scan", false, "Skip vulnerability scanning")

	return cmd
}

func writeAuditReports(cmd *cobra.Command, rep report.Report, started time.Time, jsonOutput, htmlOutput string) error {
	report.Finalize(&rep, started)
	if jsonOutput != "" {
		if err := report.WriteJSON(jsonOutput, rep); err != nil {
			return err
		}
		cmd.Printf("Wrote JSON report: %s\n", jsonOutput)
	}
	if htmlOutput != "" {
		if err := report.WriteHTML(htmlOutput, rep); err != nil {
			return err
		}
		cmd.Printf("Wrote HTML report: %s\n", htmlOutput)
	}
	return nil
}
