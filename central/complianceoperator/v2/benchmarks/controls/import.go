package controls

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

// TODO: FLOW:
// 1. Import Rules from Compliance Operator
// 2. All Controls which are not existent in ACS create a new entry or updates one
// 3. Benchmark import enriches or updates an entry

func importCISBenchmark() {
	// Open the Excel file.
	f, err := excelize.OpenFile("example.xlsx")
	if err != nil {
		log.Fatal("Failed to open Excel file:", err)
	}
	// Ensure the file is closed at the end.
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatal("Failed to close Excel file:", err)
		}
	}()

	// Get the name of the first sheet.
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		log.Fatal("Sheet name not found.")
	}

	// Get all the rows in the first sheet.
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatal("Failed to get rows:", err)
	}

	// Iterate through the rows and print each cell value.
	for _, row := range rows {
		for _, cell := range row {
			fmt.Print(cell, "\t")
		}
		fmt.Println()
	}

}
