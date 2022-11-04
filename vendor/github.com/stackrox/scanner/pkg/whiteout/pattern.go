package whiteout

var (
	// Prefix prefix means file is a whiteout. If this is followed by a
	// filename this means that file has been removed from the base layer.
	Prefix = ".wh."
	// OpaqueDirectory means the directory does not exist in lower (parent) layers,
	// and may be ignored below the current layer.
	OpaqueDirectory = ".wh..wh..opq"
)
