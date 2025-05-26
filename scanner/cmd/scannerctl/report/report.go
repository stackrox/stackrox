package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

// Cmd returns the root command for report operations.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Commands for parsing vulnerability reports",
	}
	cmd.AddCommand(vulnsCmd())
	cmd.AddCommand(diffCmd())
	return cmd
}

func printVuln(v *v4.VulnerabilityReport_Vulnerability) error {
	const vulnerabilityTemplate = `
{{.Name}}{{ if .Advisory }} ({{.Advisory}}){{end}}
----------------------
ID:          {{.Id}}
Advisory:    {{.Advisory}}
Severity:    {{.Severity}}
Description: {{.Description}}
`
	tmpl, err := template.New("vulnerability").Parse(vulnerabilityTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}
	if err := tmpl.Execute(os.Stdout, v); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}


// vulnsCmd defines the 'vulns' subcommand which accepts one report file (or uses stdin).
func vulnsCmd() *cobra.Command {
	var packageFlag string
	cmd := &cobra.Command{
		Use:   "vulns [report]",
		Short: "Parse a vulnerability report",
		Args:  cobra.MaximumNArgs(1), // If no file is provided, use stdin.
		RunE: func(cmd *cobra.Command, args []string) error {
			f := os.Stdin
			if len(args) > 0 {
				var err error
				f, err = os.Open(args[0])
				if err != nil {
					return fmt.Errorf("opening report: %w", err)
				}
			}
			report, err := parseReport(f)
			if err != nil {
				return fmt.Errorf("parse report: %w", err)
			}
			if packageFlag != "" {
				for _, pkg := range report.Contents.Packages {
					if packageFlag == pkg.Name {
						b, err := json.Marshal(pkg)
						if err != nil {
							return err
						}
						fmt.Printf("%s\n", string(b))
						for _, vId := range report.PackageVulnerabilities[pkg.Id].GetValues() {
							err := printVuln(report.Vulnerabilities[vId])
							if err != nil {
								return err
							}
						}
					}
				}
				return nil
			}
			vulnPkg := map[string][]string{}
			for pkgID, vulnIDs := range report.PackageVulnerabilities {
				for _, vID := range vulnIDs.GetValues() {
					vulnPkg[vID] = append(vulnPkg[vID], pkgID)
				}
			}
			for vID, v := range report.Vulnerabilities {
				fmt.Printf("%s\t%s\t%s\t%d\n", v.Id, v.Name, v.Advisory, len(vulnPkg[vID]))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&packageFlag, "package", "p", "", "Show vulns for a package, if affected.")
	return cmd
}

// diffCmd defines the 'diff' subcommand which accepts two positional arguments for the reports to compare.
func diffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <report1> <report2>",
		Short: "Diff two vulnerability reports",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return cmd
}

// loadReport reads the report from a file or from stdin if filename is empty.
func loadReport(filename string) ([]byte, error) {
	if filename == "" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(filename)
}

// parseReport loads and unmarshals the vulnerability report from the given filename.
func parseReport(r io.Reader) (*v4.VulnerabilityReport, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading: %w", err)
	}
	var report v4.VulnerabilityReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("unmarshaling: %w", err)
	}
	return &report, nil
}
