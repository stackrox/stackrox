package celcompile

import (
	"strings"
)

// CelPrettyPrint formats CEL code with proper indentation.
// just make it not too ugly
func CelPrettyPrint(input string) string {
	var (
		output         strings.Builder
		indentation    = 0
		indentationStr = " " // Use two spaces for each level of indentation
	)

	// Split the input code by newlines and process each line.
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		curr := indentation
		if strings.HasPrefix(trimmedLine, ".") {
			curr += 1
		}
		if strings.HasPrefix(trimmedLine, ")") {
			curr -= 1
		}
		// Add the indented line to the output.
		output.WriteString(strings.Repeat(indentationStr, 2*curr))
		output.WriteString(trimmedLine)
		output.WriteString("\n")

		indentation += 2 * strings.Count(line, "(")
		indentation -= 2 * strings.Count(line, ")")
	}
	return output.String()
}
