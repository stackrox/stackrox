package bootstrapcommon

import (
	"fmt"
	"strings"
	"unicode"
)

// GetPackageName returns the Go package name for a migration at the given start version.
func GetPackageName(startVersion int) string {
	return fmt.Sprintf("m%dtom%d", startVersion, startVersion+1)
}

// GetMigrationDirName returns the directory name for a migration.
// When zeroPad is true, version numbers are zero-padded to 3 digits.
func GetMigrationDirName(startVersion int, description string, zeroPad bool) string {
	suffix := ConvertDescriptionToSuffix(description)
	if zeroPad {
		return fmt.Sprintf("m_%03d_to_m_%03d_%s", startVersion, startVersion+1, suffix)
	}
	return fmt.Sprintf("m_%d_to_m_%d_%s", startVersion, startVersion+1, suffix)
}

// ConvertDescriptionToSuffix converts a human-readable description to a snake_case suffix.
func ConvertDescriptionToSuffix(description string) string {
	var elems []string
	var builder strings.Builder
	for _, r := range description {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(r)
		} else {
			elems = addWord(elems, &builder)
		}
	}
	elems = addWord(elems, &builder)
	return strings.Join(elems, "_")
}

func addWord(elems []string, builder *strings.Builder) []string {
	word := strings.ToLower(builder.String())
	if len(word) > 0 {
		elems = append(elems, word)
		builder.Reset()
	}
	return elems
}
