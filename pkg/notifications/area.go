package notifications

const (
	defaultArea       = "General"
	imageScanningArea = "Image Scanning"
)

var (
	moduleToArea = map[string]string{
		"reprocessor":   imageScanningArea,
		"image/service": imageScanningArea,
	}
)

// GetAreaFromModule retrieves an area based on a specific module.
func GetAreaFromModule(module string) string {
	area := moduleToArea[module]
	if area == "" {
		return defaultArea
	}
	return area
}
