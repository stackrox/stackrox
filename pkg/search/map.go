package search

import "fmt"

// ToMapKeyPath takes a path and generates a map key path
func ToMapKeyPath(path string) string {
	return fmt.Sprintf("%s.keypair.key", path)
}

// ToMapValuePath takes a path and generated a map value path
func ToMapValuePath(path string) string {
	return fmt.Sprintf("%s.keypair.value", path)
}
