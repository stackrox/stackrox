package m200tom201

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_201_to_m_202_vuln_request_v1_to_v2/schema"
	"github.com/stackrox/rox/migrator/migrations/m_201_to_m_202_vuln_request_v1_to_v2/store"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"google.golang.org/protobuf/proto"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableVulnerabilityRequestsStmt)

	return updateVulnerabilityRequests(ctx, database)
}

func updateVulnerabilityRequests(ctx context.Context, database *types.Databases) error {
	vulnReqStore := store.New(database.PostgresDB)

	var updatedObjs []*storage.VulnerabilityRequest
	var count int
	err := vulnReqStore.Walk(ctx, func(obj *storage.VulnerabilityRequest) error {

		updated := obj.CloneVT()

		// Migrate requester and approvers to new RequesterV2 and ApproversV2 fields
		if requester := obj.GetRequestor(); requester != nil {
			updated.SetRequesterV2(convertSlimUserToRequester(obj.GetRequestor()))
		}
		if approvers := obj.GetApprovers(); len(approvers) > 0 {
			updated.SetApproversV2(convertSlimUserToApprovers(obj.GetApprovers()))
		}

		// Update new expiry type field
		if obj.GetDeferralReq() != nil && obj.GetDeferralReq().GetExpiry() != nil {
			if obj.GetDeferralReq().GetExpiry().GetExpiresWhenFixed() {
				updated.GetDeferralReq().GetExpiry().SetExpiryType(storage.RequestExpiry_ANY_CVE_FIXABLE)
			} else {
				updated.GetDeferralReq().GetExpiry().SetExpiryType(storage.RequestExpiry_TIME)
			}
		}

		// Migrate UpdateDeferralReq (if any) to new DeferralUpdate field
		if obj.GetUpdatedDeferralReq().GetExpiry() != nil {
			du := &storage.DeferralUpdate{}
			du.SetCVEs(obj.GetCves().GetCves())
			du.SetExpiry(obj.GetUpdatedDeferralReq().GetExpiry())
			updated.SetDeferralUpdate(proto.ValueOrDefault(du))
			if obj.GetUpdatedDeferralReq().GetExpiry().GetExpiresWhenFixed() {
				updated.GetDeferralUpdate().GetExpiry().SetExpiryType(storage.RequestExpiry_ANY_CVE_FIXABLE)
			} else {
				updated.GetDeferralUpdate().GetExpiry().SetExpiryType(storage.RequestExpiry_TIME)
			}
		}

		// Migrate global scope to new representation for global scope
		if obj.GetScope().GetGlobalScope() != nil {
			vsi := &storage.VulnerabilityRequest_Scope_Image{}
			vsi.SetRegistry(".*")
			vsi.SetRemote(".*")
			vsi.SetTag(".*")
			vs := &storage.VulnerabilityRequest_Scope{}
			vs.SetImageScope(proto.ValueOrDefault(vsi))
			updated.SetScope(vs)
		}

		updatedObjs = append(updatedObjs, updated)
		count++

		if len(updatedObjs) == batchSize {
			err := vulnReqStore.UpsertMany(ctx, updatedObjs)
			if err != nil {
				return errors.Wrapf(err, "failed to write updated records to %s table", schema.VulnerabilityRequestsTableName)
			}
			updatedObjs = updatedObjs[:0]
		}
		return nil
	})

	if err != nil {
		return errors.Wrapf(err, "failed to update %s table", schema.VulnerabilityRequestsTableName)
	}

	if len(updatedObjs) > 0 {
		err := vulnReqStore.UpsertMany(ctx, updatedObjs)
		if err != nil {
			return errors.Wrapf(err, "failed to write updated records to %s table", schema.VulnerabilityRequestsTableName)
		}
	}
	log.Infof("Updated %d vulnerability exceptions", count)
	return nil
}

func convertSlimUserToRequester(user *storage.SlimUser) *storage.Requester {
	if user == nil {
		return nil
	}
	requester := &storage.Requester{}
	requester.SetId(user.GetId())
	requester.SetName(user.GetName())
	return requester
}

func convertSlimUserToApprovers(users []*storage.SlimUser) []*storage.Approver {
	approvers := make([]*storage.Approver, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		approver := &storage.Approver{}
		approver.SetId(user.GetId())
		approver.SetName(user.GetName())
		approvers = append(approvers, approver)
	}
	if len(approvers) == 0 {
		return nil
	}
	return approvers
}
