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
			updated.RequesterV2 = convertSlimUserToRequester(obj.GetRequestor())
		}
		if approvers := obj.GetApprovers(); len(approvers) > 0 {
			updated.ApproversV2 = convertSlimUserToApprovers(obj.GetApprovers())
		}

		// Update new expiry type field
		if obj.GetDeferralReq() != nil && obj.GetDeferralReq().GetExpiry() != nil {
			if obj.GetDeferralReq().GetExpiry().GetExpiresWhenFixed() {
				updated.GetDeferralReq().Expiry.ExpiryType = storage.RequestExpiry_ANY_CVE_FIXABLE
			} else {
				updated.GetDeferralReq().Expiry.ExpiryType = storage.RequestExpiry_TIME
			}
		}

		// Migrate UpdateDeferralReq (if any) to new DeferralUpdate field
		if obj.GetUpdatedDeferralReq().GetExpiry() != nil {
			updated.UpdatedReq = &storage.VulnerabilityRequest_DeferralUpdate{
				DeferralUpdate: &storage.DeferralUpdate{
					CVEs:   obj.GetCves().GetCves(),
					Expiry: obj.GetUpdatedDeferralReq().GetExpiry(),
				},
			}
			if obj.GetUpdatedDeferralReq().GetExpiry().GetExpiresWhenFixed() {
				updated.GetDeferralUpdate().Expiry.ExpiryType = storage.RequestExpiry_ANY_CVE_FIXABLE
			} else {
				updated.GetDeferralUpdate().Expiry.ExpiryType = storage.RequestExpiry_TIME
			}
		}

		// Migrate global scope to new representation for global scope
		if obj.GetScope().GetGlobalScope() != nil {
			updated.Scope = &storage.VulnerabilityRequest_Scope{
				Info: &storage.VulnerabilityRequest_Scope_ImageScope{
					ImageScope: &storage.VulnerabilityRequest_Scope_Image{
						Registry: ".*",
						Remote:   ".*",
						Tag:      ".*",
					},
				},
			}
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
	}, true)

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
	return &storage.Requester{
		Id:   user.GetId(),
		Name: user.GetName(),
	}
}

func convertSlimUserToApprovers(users []*storage.SlimUser) []*storage.Approver {
	approvers := make([]*storage.Approver, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		approvers = append(approvers, &storage.Approver{
			Id:   user.GetId(),
			Name: user.GetName(),
		})
	}
	if len(approvers) == 0 {
		return nil
	}
	return approvers
}
