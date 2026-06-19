package bootstrapcommon

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
)

// RegisterMigration adds a blank import for the migration package to the registration file.
// It handles both proper import blocks and commented-out import blocks.
func RegisterMigration(registrationFilePath, migrationDirName, registrationPrefix string) error {
	newFileLines, err := buildRegistrationLines(registrationFilePath, migrationDirName, registrationPrefix)
	if err != nil {
		return err
	}
	writeFile, err := os.OpenFile(registrationFilePath, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer closeFile(writeFile)
	for _, line := range newFileLines {
		fmt.Fprintln(writeFile, line)
	}
	return nil
}

func buildRegistrationLines(registrationFilePath, migrationDirName, registrationPrefix string) ([]string, error) {
	readFile, err := os.Open(registrationFilePath)
	if err != nil {
		return nil, err
	}
	defer closeFile(readFile)

	registeredPath := path.Join(registrationPrefix, migrationDirName)
	importLine := fmt.Sprintf("\t_ %q", registeredPath)

	// First pass: try to insert into an existing import block.
	var newFileLines []string
	fileScanner := bufio.NewScanner(readFile)
	inImportBlock := false
	registered := false

	for fileScanner.Scan() {
		line := fileScanner.Text()

		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
		}

		if inImportBlock && strings.HasPrefix(line, ")") {
			if !registered {
				newFileLines = append(newFileLines, importLine)
				registered = true
			}
			inImportBlock = false
		}

		newFileLines = append(newFileLines, line)
	}

	if registered {
		return newFileLines, nil
	}

	// Second pass: no import block found — rewrite commented-out imports into a real block.
	if _, err := readFile.Seek(0, 0); err != nil {
		return nil, err
	}
	fileScanner = bufio.NewScanner(readFile)
	newFileLines = nil
	wroteImport := false

	for fileScanner.Scan() {
		line := fileScanner.Text()

		if strings.HasPrefix(line, "// import (") || strings.HasPrefix(line, "// Import ") {
			if !wroteImport {
				newFileLines = append(newFileLines,
					"// Import migration packages here to register them via init().",
					"import (",
					importLine,
					")",
				)
				wroteImport = true
			}
			continue
		}
		if strings.HasPrefix(line, "// \t") || line == "// )" {
			continue
		}

		newFileLines = append(newFileLines, line)
	}

	if !wroteImport {
		return nil, fmt.Errorf("failed to register migration in %s", registrationFilePath)
	}

	return newFileLines, nil
}

func closeFile(file *os.File) {
	if err := file.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Error closing file %q\n", file.Name())
	}
}
