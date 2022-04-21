package service

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/license/manager"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Licenses)): {
			"/v1.LicenseService/GetLicenses",
		},
		user.With(permissions.Modify(resources.Licenses)): {
			"/v1.LicenseService/AddLicense",
		},
		user.With(): {
			"/v1.LicenseService/GetActiveLicenseExpiration",
		},
		idcheck.ScannerOnly(): {
			"/v1.LicenseService/GetActiveLicenseKey",
		},
	})
)

type service struct {
	lockdownMode bool

	licenseMgr manager.LicenseManager
}

func newService(lockdownMode bool, licenseMgr manager.LicenseManager) *service {
	return &service{
		lockdownMode: lockdownMode,
		licenseMgr:   licenseMgr,
	}
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterLicenseServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLicenseServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	if s.lockdownMode {
		return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
	}
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetActiveLicenseKey(context.Context, *v1.Empty) (*v1.GetActiveLicenseKeyResponse, error) {
	if s.lockdownMode {
		return nil, status.Error(codes.Unavailable, "this API is not available without an active license")
	}
	return &v1.GetActiveLicenseKeyResponse{LicenseKey: s.licenseMgr.GetActiveLicenseKey()}, nil
}

func (s *service) GetLicenses(ctx context.Context, req *v1.GetLicensesRequest) (*v1.GetLicensesResponse, error) {
	allLicenseInfos := s.licenseMgr.GetAllLicenses()

	var selected []*v1.LicenseInfo
	for _, licenseInfo := range allLicenseInfos {
		if req.GetActiveOpt() != nil {
			if req.GetActive() != licenseInfo.GetActive() {
				continue
			}
		}
		if len(req.GetStatuses()) != 0 && sliceutils.Find(req.GetStatuses(), licenseInfo.GetStatus()) == -1 {
			continue
		}

		selected = append(selected, licenseInfo)
	}

	resp := &v1.GetLicensesResponse{
		Licenses: selected,
	}
	return resp, nil
}

var (
	licenseAcceptedStates = map[v1.LicenseInfo_Status]struct{}{
		v1.LicenseInfo_NOT_YET_VALID: {},
		v1.LicenseInfo_VALID:         {},
	}
)

func (s *service) AddLicense(ctx context.Context, req *v1.AddLicenseRequest) (*v1.AddLicenseResponse, error) {
	if req.GetLicenseKey() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "must provide a non-empty license key")
	}

	licenseInfo, err := s.licenseMgr.AddLicenseKey(req.GetLicenseKey(), req.GetActivate())
	if err != nil {
		return nil, errors.Errorf("failed to add license key: %v", err)
	}

	_, licenseAccepted := licenseAcceptedStates[licenseInfo.GetStatus()]

	return &v1.AddLicenseResponse{
		License:  licenseInfo,
		Accepted: licenseAccepted,
	}, nil
}

func (s *service) GetActiveLicenseExpiration(ctx context.Context, _ *v1.Empty) (*v1.GetActiveLicenseExpirationResponse, error) {
	if s.lockdownMode {
		return nil, status.Error(codes.Unavailable, "this API is not available without an active license")
	}

	activeLicense := s.licenseMgr.GetActiveLicense()
	expirationTS := activeLicense.GetRestrictions().GetNotValidAfter()
	if expirationTS == nil {
		expirationTS = types.TimestampNow()
	}

	return &v1.GetActiveLicenseExpirationResponse{
		ExpirationTime: expirationTS,
	}, nil
}
