package standards

import "fmt"

func buildQualifiedID(standardID, controlOrCategoryID string) string {
	return fmt.Sprintf("%s:%s", standardID, controlOrCategoryID)
}
