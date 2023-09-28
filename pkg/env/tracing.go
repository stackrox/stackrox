package env

// Open Telemetry tracing

// OpenTelemetryCollectorURL is the open telemetry collector host address.
var OpenTelemetryCollectorURL = RegisterSetting("ROX_OTEL_COLLECTOR_URL", WithDefault("jaeger-collector.default.svc.cluster.local.:4317"))
