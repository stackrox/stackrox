// Package config_controller_ca contains test certificates for config-controller additional CA testing.
package config_controller_ca

import _ "embed"

// RootCACert is the root CA certificate used to sign the server certificate.
// This CA should be added to the additional-ca secret for config-controller to trust.
//
//go:embed root-ca.crt
var RootCACert []byte

// ServerCert is the server certificate with SANs for nginx-proxy.qa-config-controller-ca.
//
//go:embed server.crt
var ServerCert []byte

// ServerKey is the private key for the server certificate.
//
//go:embed server.key
var ServerKey []byte
