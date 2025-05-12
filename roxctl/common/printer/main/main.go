package main

import (
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

func main() {
	data := [][]string{
		{"Package", "Version", "Status"},
		{"table\nwriter", "v0.0.5", "legacy"},
		{"table\nwriter", "v1.0.0", "latest"},
	}

	table := tablewriter.NewTable(os.Stdout,
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Settings: tw.Settings{
				Separators: tw.Separators{BetweenRows: tw.On},
				Lines:      tw.Lines{ShowFooterLine: tw.On},
			},
		})),
		tablewriter.WithConfig(tablewriter.Config{
			Header: tw.CellConfig{
				Formatting: tw.CellFormatting{Alignment: tw.AlignCenter},
			},
			Row: tw.CellConfig{
				Formatting: tw.CellFormatting{
					MergeMode: tw.MergeHierarchical,
					Alignment: tw.AlignCenter,
					AutoWrap:  tw.WrapNone,
				},
			},
		}),
	)

	table.Header(data[0])
	table.Bulk(data[1:])
	table.Render()
}
