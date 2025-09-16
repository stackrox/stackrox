package certrefresh

// Package certrefresh handles TLS certificate lifecycle management for Sensor components.
//
// This package provides functionality for:
// - Automatic certificate renewal before expiration
// - Certificate repository management for persistent storage
// - CA bundle management for admission control webhooks
//
// ## Certificate Refresh Flow
//
// The main certificate refresh process periodically requests fresh certificates
// from Central and stores them in Kubernetes secrets for use by Sensor components.
//
// ## CA Bundle Management
//
// The CA bundle ConfigMap contains the internal CA certificates trusted by
// Central and is updated through two flows:
//
// 1. TLS Challenge Flow:
//    - Occurs during Sensor startup for Operator-managed clusters
//    - Retrieves current CA certificates via TLS challenge with Central
//
// 2. Certificate Refresh Flow:
//    - Occurs during periodic certificate refresh operations
//    - Receives fresh CA bundle data via CaBundlePem field from Central
//
// Both flows ensure admission control webhooks have up-to-date CA certificates
// during CA rotation. The Operator is watching the CA bundle ConfigMap and
// updates the CA bundle field in the ValidatingWebhookConfiguration CR when
// the ConfigMap is updated.
