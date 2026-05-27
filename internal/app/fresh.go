package app

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/m11s-io/zick/internal/cli"
	"github.com/m11s-io/zick/internal/config"
	"github.com/m11s-io/zick/internal/fresh"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newFreshCmd() *cobra.Command {
	var ageDays int
	var failOn string
	var includeDev bool
	var format string

	cmd := &cobra.Command{
		Use:     "fresh [path]",
		Short:   "Check dependencies for supply chain risk (freshness age gate)",
		GroupID: "scan",
		Long: `Queries package registries for publish timestamps and flags dependencies
published within the configured age gate. Helps catch supply chain attacks
before packages are installed.

Reads bun.lock, pnpm-lock.yaml, yarn.lock, package-lock.json, or package.json.`,
		Example: `  # check current directory with the default 7-day gate
  zick fresh .

  # stricter gate, include devDependencies, JSON output
  zick fresh --age-gate 3 --include-dev --format json .

  # exit 1 on any package below the warn threshold
  zick fresh --fail-on warn .`,
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
			if !flagChanged(cmd, "age-gate") && cfg.Fresh.AgeGateDays != nil {
				ageDays = *cfg.Fresh.AgeGateDays
			}
			if !flagChanged(cmd, "include-dev") && cfg.Fresh.IncludeDev != nil {
				includeDev = *cfg.Fresh.IncludeDev
			}
			if !flagChanged(cmd, "fail-on") && cfg.Fresh.FailOn != "" {
				failOn = cfg.Fresh.FailOn
			}
			if !flagChanged(cmd, "format") && cfg.Fresh.Format != "" {
				format = cfg.Fresh.Format
			}
			if err := validateFreshFlags(ageDays, failOn, format); err != nil {
				return err
			}

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
				return nil
			}

			if err := printFreshResults(cmd, results, format); err != nil {
				return err
			}

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
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json)")

	return cmd
}

func flagChanged(cmd *cobra.Command, name string) bool {
	changed := false
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if flag.Name == name {
			changed = true
		}
	})
	return changed
}

func validateFreshFlags(ageDays int, failOn, format string) error {
	if ageDays <= 0 {
		return fmt.Errorf("--age-gate must be greater than 0")
	}
	switch failOn {
	case "high", "warn":
	default:
		return fmt.Errorf("--fail-on must be one of: high, warn")
	}
	switch format {
	case "table", "json":
		return nil
	default:
		return fmt.Errorf("--format must be one of: table, json")
	}
}

func printFreshResults(cmd *cobra.Command, results []fresh.Result, format string) error {
	if format == "json" {
		return printFreshJSON(cmd, results)
	}
	printFreshTable(cmd, results)
	return nil
}

type freshJSONResult struct {
	Risk      string `json:"risk"`
	Package   string `json:"package"`
	Version   string `json:"version"`
	Published string `json:"published"`
	Age       string `json:"age"`
}

func printFreshJSON(cmd *cobra.Command, results []fresh.Result) error {
	rows := make([]freshJSONResult, 0, len(results))
	for _, r := range results {
		rows = append(rows, freshJSONResult{
			Risk:      riskLabel(r.Risk),
			Package:   r.Package,
			Version:   r.Version,
			Published: r.Published.Format(time.RFC3339),
			Age:       humanAge(r.Age),
		})
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
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
