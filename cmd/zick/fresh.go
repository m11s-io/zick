package main

import (
	"fmt"
	"time"

	"github.com/m11s-io/zick/internal/cli"
	"github.com/m11s-io/zick/internal/fresh"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func newFreshCmd() *cobra.Command {
	var ageDays int
	var failOn string
	var includeDev bool

	cmd := &cobra.Command{
		Use:     "fresh [path]",
		Short:   "Check dependencies for supply chain risk (freshness age gate)",
		GroupID: "scan",
		Long: `Queries package registries for publish timestamps and flags dependencies
published within the configured age gate. Helps catch supply chain attacks
before packages are installed.

Reads package-lock.json (exact versions) or package.json (latest from registry).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			results, err := fresh.Check(path, fresh.Options{
				AgeDays:    ageDays,
				IncludeDev: includeDev,
			})
			if err != nil {
				return err
			}

			if len(results) == 0 {
				cmd.Println("No supported manifest found (package-lock.json, package.json).")
				return nil
			}

			printFreshTable(cmd, results)

			violations := countViolations(results, failOn)
			if violations > 0 {
				cmd.PrintErrf("\n%d package(s) below the %d-day age gate.\n", violations, ageDays)
				return &cli.SilentError{Code: 1}
			}

			cmd.Printf("\nAll packages pass the %d-day age gate.\n", ageDays)
			return nil
		},
	}

	cmd.Flags().IntVar(&ageDays, "age-gate", 7, "Flag packages published within this many days")
	cmd.Flags().StringVar(&failOn, "fail-on", "high", "Exit 1 when this risk level is found (high, warn)")
	cmd.Flags().BoolVar(&includeDev, "include-dev", false, "Include devDependencies")

	return cmd
}

func printFreshTable(cmd *cobra.Command, results []fresh.Result) {
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"RISK", "PACKAGE", "VERSION", "PUBLISHED", "AGE"})
	table.SetBorder(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetColumnSeparator("  ")
	table.SetHeaderLine(false)

	for _, r := range results {
		table.Append([]string{
			riskLabel(r.Risk),
			r.Package,
			r.Version,
			r.Published.Format("2006-01-02"),
			humanAge(r.Age),
		})
	}

	table.Render()
}

func riskLabel(r fresh.Risk) string {
	switch r {
	case fresh.RiskHigh:
		return "HIGH"
	case fresh.RiskWarn:
		return "WARN"
	default:
		return "OK"
	}
}

func countViolations(results []fresh.Result, failOn string) int {
	count := 0
	for _, r := range results {
		switch failOn {
		case "warn":
			if r.Risk >= fresh.RiskWarn {
				count++
			}
		default: // "high"
			if r.Risk >= fresh.RiskHigh {
				count++
			}
		}
	}
	return count
}

func humanAge(d time.Duration) string {
	hours := d.Hours()
	switch {
	case hours < 24:
		return fmt.Sprintf("%.0f hours ago", hours)
	case hours < 24*7:
		return fmt.Sprintf("%.0f days ago", hours/24)
	case hours < 24*30:
		return fmt.Sprintf("%.0f weeks ago", hours/(24*7))
	case hours < 24*365:
		return fmt.Sprintf("%.0f months ago", hours/(24*30))
	default:
		return fmt.Sprintf("%.0f years ago", hours/(24*365))
	}
}
