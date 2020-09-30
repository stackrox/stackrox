package derivelocalvalues

import (
	"fmt"
	"strings"
)

func extractImageRegistry(imageFullRef string, imageName string) *string {
	if imageFullRef == "" {
		return nil
	}
	registry := ""
	components := strings.Split(imageFullRef, fmt.Sprintf("/%s:", imageName))
	if len(components) == 0 {
		return nil
	}
	registry = components[0]
	return &registry
}
