package internal

import "github.com/stackrox/rox/pkg/grpc/authz"

// Authorizer is the pre-approved per-RPC authorizer for the external backups service.
var Authorizer authz.Authorizer
