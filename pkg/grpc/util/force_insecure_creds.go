package util

import "google.golang.org/grpc/credentials"

type forcedInsecureCreds struct {
	credentials.PerRPCCredentials
}

func (c *forcedInsecureCreds) RequireTransportSecurity() bool {
	return false
}

// ForceInsecureCreds returns a version of creds that DO NOT require transport security.
func ForceInsecureCreds(creds credentials.PerRPCCredentials) credentials.PerRPCCredentials {
	if creds.RequireTransportSecurity() {
		creds = &forcedInsecureCreds{PerRPCCredentials: creds}
	}
	return creds
}
