package repositorytocpe

import (
	"encoding/json"
	"fmt"

	"github.com/cespare/xxhash/v2"
)

// MaxMappingBytes is the accepted size cap for repository-to-CPE JSON (5 MiB).
const MaxMappingBytes = 5 * 1024 * 1024

// ValidateMapping rejects oversize or undecodable repository-to-CPE JSON.
func ValidateMapping(content []byte) error {
	if len(content) > MaxMappingBytes {
		return fmt.Errorf("mapping size %d exceeds %d bytes", len(content), MaxMappingBytes)
	}
	var m MappingFile
	if err := json.Unmarshal(content, &m); err != nil {
		return fmt.Errorf("decode mapping: %w", err)
	}
	if m.Data == nil {
		return fmt.Errorf("mapping missing data object")
	}
	return nil
}

// HashMapping returns XXH64 of content as 16 lowercase hex characters.
func HashMapping(content []byte) string {
	sum := xxhash.Sum64(content)
	return fmt.Sprintf("%016x", sum)
}
