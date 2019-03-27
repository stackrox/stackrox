package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/uuid"
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
	})

	customerID = uuid.NewV4().String()

	licenseInfos = []*v1.LicenseInfo{
		{
			License: &v1.License{
				Metadata: &v1.License_Metadata{
					Id:              uuid.NewV4().String(),
					SigningKeyId:    "test/key/1",
					IssueDate:       protoconv.ConvertTimeToTimestamp(time.Now().Add(-42 * 24 * time.Hour)),
					LicensedForId:   customerID,
					LicensedForName: "Acme Inc.",
				},
				SupportContact: &v1.License_Contact{
					Phone: "+1 (123) 456-7890",
					Email: "support@stackrox.com",
					Url:   "https://stackrox.com",
					Name:  "StackRox Customer Support",
				},
				Restrictions: &v1.License_Restrictions{
					NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-42 * 24 * time.Hour)),
					NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(-14 * 24 * time.Hour)),
					AllowOffline:                       true,
					MaxNodes:                           100,
					NoBuildFlavorRestriction:           true,
					NoDeploymentEnvironmentRestriction: true,
				},
			},
			Status: v1.LicenseInfo_EXPIRED,
			Active: false,
		},
		{
			License: &v1.License{
				Metadata: &v1.License_Metadata{
					Id:              uuid.NewV4().String(),
					SigningKeyId:    "test/key/1",
					IssueDate:       protoconv.ConvertTimeToTimestamp(time.Now().Add(-14 * 24 * time.Hour)),
					LicensedForId:   customerID,
					LicensedForName: "Acme Inc.",
				},
				SupportContact: &v1.License_Contact{
					Phone: "+1 (123) 456-7890",
					Email: "support@stackrox.com",
					Url:   "https://stackrox.com",
					Name:  "StackRox Customer Support",
				},
				Restrictions: &v1.License_Restrictions{
					NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-14 * 24 * time.Hour)),
					NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(28 * 24 * time.Hour)),
					AllowOffline:                       true,
					MaxNodes:                           100,
					NoBuildFlavorRestriction:           true,
					NoDeploymentEnvironmentRestriction: true,
				},
			},
			Status: v1.LicenseInfo_VALID,
			Active: true,
		},
		{
			License: &v1.License{
				Metadata: &v1.License_Metadata{
					Id:              uuid.NewV4().String(),
					SigningKeyId:    "test/key/1",
					IssueDate:       protoconv.ConvertTimeToTimestamp(time.Now()),
					LicensedForId:   customerID,
					LicensedForName: "Acme Inc.",
				},
				SupportContact: &v1.License_Contact{
					Phone: "+1 (123) 456-7890",
					Email: "support@stackrox.com",
					Url:   "https://stackrox.com",
					Name:  "StackRox Customer Support",
				},
				Restrictions: &v1.License_Restrictions{
					NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(14 * 24 * time.Hour)),
					NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(42 * 24 * time.Hour)),
					AllowOffline:                       true,
					MaxNodes:                           100,
					NoBuildFlavorRestriction:           true,
					NoDeploymentEnvironmentRestriction: true,
				},
			},
			Status: v1.LicenseInfo_NOT_YET_VALID,
			Active: false,
		},
	}
)

type service struct {
}

func newService() *service {
	return &service{}
}

func (s *service) RegisterServiceServer(server *grpc.Server) {
	v1.RegisterLicenseServiceServer(server, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterLicenseServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetLicenses(ctx context.Context, req *v1.GetLicensesRequest) (*v1.GetLicensesResponse, error) {
	selected := make([]*v1.LicenseInfo, 0, len(licenseInfos))

	for _, licenseInfo := range licenseInfos {
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

func (s *service) AddLicense(ctx context.Context, req *v1.AddLicenseRequest) (*v1.AddLicenseResponse, error) {
	if req.GetLicenseKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "must provide a non-empty license key")
	}

	fakeLicense := &v1.LicenseInfo{
		License: &v1.License{
			Metadata: &v1.License_Metadata{
				Id:              uuid.NewV4().String(),
				SigningKeyId:    "test/key/1",
				IssueDate:       protoconv.ConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				LicensedForId:   customerID,
				LicensedForName: "Acme Inc.",
			},
			SupportContact: &v1.License_Contact{
				Phone: "+1 (123) 456-7890",
				Email: "support@stackrox.com",
				Url:   "https://stackrox.com",
				Name:  "StackRox Customer Support",
			},
			Restrictions: &v1.License_Restrictions{
				NotValidBefore:                     protoconv.ConvertTimeToTimestamp(time.Now().Add(-1 * time.Hour)),
				NotValidAfter:                      protoconv.ConvertTimeToTimestamp(time.Now().Add(24 * time.Hour)),
				AllowOffline:                       true,
				MaxNodes:                           100,
				NoBuildFlavorRestriction:           true,
				NoDeploymentEnvironmentRestriction: true,
			},
		},
		Status:       v1.LicenseInfo_OTHER,
		StatusReason: "The license was signed with an invalid key. Please contact support if you believe this is an error.",
		Active:       false,
	}

	return &v1.AddLicenseResponse{
		License:  fakeLicense,
		Accepted: false,
	}, nil
}
