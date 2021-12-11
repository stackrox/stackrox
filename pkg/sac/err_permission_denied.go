package sac

import (
	"github.com/stackrox/rox/pkg/errorhelpers"
	"google.golang.org/grpc/codes"
)

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errorhelpers.NewWithGRPCCode(codes.PermissionDenied, "access to resource denied")
)
