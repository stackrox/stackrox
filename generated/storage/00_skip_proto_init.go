package storage

import "os"

// skipProtoInit allows skipping proto type registry initialization at runtime.
// Set ROX_SKIP_PROTO_INIT=true for sensor/AC/CC entrypoints where vtprotobuf
// handles all serialization without the global registry. Central and CLI tools
// that use protojson/reflection should NOT set this variable.
// Saves ~10-15 MB of heap at startup.
var skipProtoInit = os.Getenv("ROX_SKIP_PROTO_INIT") == "true"
