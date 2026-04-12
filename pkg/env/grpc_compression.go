package env

// SensorGRPCCompression controls whether sensor uses gzip compression
// for gRPC communication with central. Compression saves network bandwidth
// but costs ~3 MB of memory for compression buffers and ongoing CPU for
// compression/decompression. On local/same-cluster networks, disabling
// compression reduces memory and CPU usage.
var SensorGRPCCompression = RegisterBooleanSetting("ROX_SENSOR_GRPC_COMPRESSION", false)
