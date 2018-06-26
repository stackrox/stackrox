package mappings

// ObjectMap is the object string field mapping for alerts.
var ObjectMap = map[string]string{
	"image":      "alert.deployment.containers.image",
	"deployment": "alert.deployment",
	"policy":     "alert.policy",
}
