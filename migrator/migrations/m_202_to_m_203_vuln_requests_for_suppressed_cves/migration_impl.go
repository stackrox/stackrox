package m201tom202

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_202_to_m_203_vuln_requests_for_suppressed_cves/schema"
	vulnReqStore "github.com/stackrox/rox/migrator/migrations/m_202_to_m_203_vuln_requests_for_suppressed_cves/store/vulnerabilityrequests"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm/clause"
)

var (
	batchSize = 2000
	log       = logging.LoggerForModule()

	// UUID namespace for all the UUID created in this migration.
	systemGeneratedUUIDNS = uuid.FromStringOrNil("a9c8adaf-d9c7-490f-afe0-7d05f7327982")
	userID                = uuid.NewV4().String()
	sysUser               = &storage.SlimUser{
		Id:   userID,
		Name: "system",
	}
	sysRequester = &storage.Requester{
		Id:   userID,
		Name: "system",
	}
	sysApprover = &storage.Approver{
		Id:   userID,
		Name: "system",
	}
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableImagesStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableImageCvesStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableImageCveEdgesStmt)
	pgutils.CreateTableFromModel(ctx, database.GormDB, schema.CreateTableVulnerabilityRequestsStmt)

	now := protocompat.TimestampNow()
	snoozedCVEMap, err := collectSnoozedImageCVEs(ctx, database)
	if err != nil || len(snoozedCVEMap) == 0 {
		return err
	}

	if err = createVulnRequests(ctx, database, now, snoozedCVEMap); err != nil {
		return err
	}
	// Note that the fields of `storage.ImageCVE` used by 1st generation exception workflow must not be reset
	// for backward compatibility purpose.
	return updateImageCVEEdges(ctx, database, snoozedCVEMap)
}

func collectSnoozedImageCVEs(ctx context.Context, database *types.Databases) (map[string]*protocompat.Timestamp, error) {
	query := database.GormDB.WithContext(ctx).Table(schema.ImageCvesTableName).
		Select("serialized").Where("snoozed = ?", "true")
	rows, err := query.Rows()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query the table %s", schema.ImageCvesTableName)
	}
	defer func() { _ = rows.Close() }()

	// Map of CVE to expiry.
	cveMap := make(map[string]*protocompat.Timestamp)
	var count int
	for rows.Next() {
		var obj schema.ImageCves
		if err = query.ScanRows(rows, &obj); err != nil {
			return nil, errors.Wrap(err, "failed to scan 'image_cves' table records")
		}
		proto, err := schema.ConvertImageCVEToProto(&obj)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert %+v to proto", proto)
		}
		// Just to be sure
		if !proto.GetSnoozed() {
			continue
		}

		if proto.GetCveBaseInfo() != nil {
			cveMap[proto.GetCveBaseInfo().GetCve()] = proto.GetSnoozeExpiry()
			count++
		}
	}
	if rows.Err() != nil {
		return nil, errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.ImageCvesTableName)
	}

	log.Infof("Found %d globally snoozed vulnerabilities that need to be migrated", count)
	return cveMap, nil
}

func createVulnRequests(ctx context.Context, database *types.Databases, now *protocompat.Timestamp, cveMap map[string]*protocompat.Timestamp) error {
	store := vulnReqStore.New(database.PostgresDB)

	var vulnReqs []*storage.VulnerabilityRequest
	var count int
	for cve, expiry := range cveMap {
		// If the snooze expiry is past due, no need to create v2 exceptions since those will be reverted on Central startup anyway.
		if expiry != nil && protocompat.CompareTimestamps(now, expiry) >= 0 {
			continue
		}

		// We do not allow multiple exceptions covering the same scope.
		// The following check will return true in case of upgrade-rollback-upgrade path either because of:
		// - previous migration run, or
		// - upgrade -> v2 request created -> rollback -> same v1 snooze created -> upgrade
		matchingReqsExist, err := checkMatchingRequestsExist(ctx, database, cve)
		if err != nil {
			return errors.Wrapf(err, "failed to check if global vulnerability exception for CVE %s exist in the database", cve)
		}
		if matchingReqsExist {
			log.Infof("global vulnerability exception already exists for CVE %s", cve)
			continue
		}

		vulnReqs = append(vulnReqs, createVulnerabilityRequest(cve, now, expiry))
		count++

		if len(vulnReqs) == batchSize {
			if err := store.UpsertMany(ctx, vulnReqs); err != nil {
				return errors.Wrapf(err, "failed to upsert %d vulnerability exceptions after %d were upserted", len(vulnReqs), count-len(vulnReqs))
			}
			vulnReqs = vulnReqs[:0]
		}
	}

	if len(vulnReqs) > 0 {
		if err := store.UpsertMany(ctx, vulnReqs); err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(vulnReqs))
		}
	}
	log.Infof("Created %d vulnerability exceptions for globally snoozed vulnerabilities", count)
	return nil
}

func updateImageCVEEdges(ctx context.Context, database *types.Databases, cveMap map[string]*protocompat.Timestamp) error {
	cves := make([]string, 0, len(cveMap))
	for cve := range cveMap {
		cves = append(cves, cve)
	}

	db := database.GormDB.WithContext(ctx).Table(schema.ImageCveEdgesTableName)
	query := database.GormDB.WithContext(ctx).Table(schema.ImageCveEdgesTableName).
		Raw(fmt.Sprintf("SELECT %[1]s.serialized FROM %[1]s WHERE EXISTS (SELECT 1 FROM %[2]s WHERE %[1]s.imagecveid = %[2]s.id AND %[2]s.cvebaseinfo_cve = ANY($1::text[]))", schema.ImageCveEdgesTableName, schema.ImageCvesTableName), cves)
	rows, err := query.Rows()
	if err != nil {
		return errors.Wrapf(err, "failed to query table %s", schema.ImageCvesTableName)
	}
	defer func() { _ = rows.Close() }()

	var convertedEdges []*schema.ImageCveEdges
	var count int
	for rows.Next() {
		var obj schema.ImageCveEdges
		if err = query.ScanRows(rows, &obj); err != nil {
			return errors.Wrap(err, "failed to scan 'image_cve_edges' table rows")
		}
		proto, err := schema.ConvertImageCVEEdgeToProto(&obj)
		if err != nil {
			return errors.Wrapf(err, "failed to convert %+v to proto", proto)
		}

		proto.SetState(storage.VulnerabilityState_DEFERRED)

		converted, err := schema.ConvertImageCVEEdgeFromProto(proto)
		if err != nil {
			return errors.Wrapf(err, "failed to convert from proto %+v", proto)
		}
		convertedEdges = append(convertedEdges, converted)
		count++

		if len(cves) == batchSize {
			if err = db.
				Clauses(clause.OnConflict{UpdateAll: true}).
				Model(schema.CreateTableImageCveEdgesStmt.GormModel).
				Create(&convertedEdges).Error; err != nil {
				return errors.Wrapf(err, "failed to upsert converted %d objects after %d upserted", len(cves), count-len(cves))
			}
			cves = cves[:0]
		}
	}
	if rows.Err() != nil {
		return errors.Wrapf(rows.Err(), "failed to get rows for %s", schema.ImageCvesTableName)
	}

	if len(convertedEdges) > 0 {
		if err = db.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Model(schema.CreateTableImageCveEdgesStmt.GormModel).
			Create(&convertedEdges).Error; err != nil {
			return errors.Wrapf(err, "failed to upsert last %d objects", len(cves))
		}
	}
	log.Infof("Updated %d 'image_cve_edges' records", count)
	return nil
}

func checkMatchingRequestsExist(ctx context.Context, database *types.Databases, cve string) (bool, error) {
	query := database.GormDB.WithContext(ctx).Table(schema.VulnerabilityRequestsTableName).
		Where("? = ANY(cves_cves)", cve).
		Where(schema.VulnerabilityRequests{ScopeImageScopeRegistry: ".*"}).
		Where(schema.VulnerabilityRequests{ScopeImageScopeRemote: ".*"}).
		Where(schema.VulnerabilityRequests{ScopeImageScopeTag: ".*"}).
		Where(schema.VulnerabilityRequests{Expired: false})
	var count int64
	tx := query.Count(&count)
	if tx.Error != nil {
		return false, tx.Error
	}
	return count > 0, nil
}

func createVulnerabilityRequest(cve string, now, expiry *protocompat.Timestamp) *storage.VulnerabilityRequest {
	return storage.VulnerabilityRequest_builder{
		Id:          exceptionID(cve),
		Name:        exceptionName(cve),
		TargetState: storage.VulnerabilityState_DEFERRED,
		Status:      storage.RequestStatus_APPROVED,
		Expired:     false,
		Requestor:   sysUser,
		Approvers:   []*storage.SlimUser{sysUser},
		RequesterV2: sysRequester,
		ApproversV2: []*storage.Approver{sysApprover},
		CreatedAt:   now,
		LastUpdated: now,
		Comments: []*storage.RequestComment{
			storage.RequestComment_builder{
				Id:        uuid.NewV4().String(),
				Message:   "This is a system-generated exception for legacy global vulnerability deferral found during system upgrade",
				User:      sysUser,
				CreatedAt: now,
			}.Build(),
		},
		Scope: storage.VulnerabilityRequest_Scope_builder{
			ImageScope: storage.VulnerabilityRequest_Scope_Image_builder{
				Registry: ".*",
				Remote:   ".*",
				Tag:      ".*",
			}.Build(),
		}.Build(),
		DeferralReq: proto.ValueOrDefault(getDeferralRequest(expiry)),
		Cves: storage.VulnerabilityRequest_CVEs_builder{
			Cves: []string{cve},
		}.Build(),
	}.Build()
}

// This guarantees unique exception name because CVE is unique. This approach avoids extra database lookup
// required to avoid sequence number conflict in ABC-YYMMDD-SEQNUM pattern.
func exceptionName(cve string) string {
	return strings.ReplaceAll(strings.ToUpper(cve), "CVE", "SYS")
}

// Creates a deterministic UUID from CVE string the specified namespace.
func exceptionID(cve string) string {
	return uuid.NewV5(systemGeneratedUUIDNS, exceptionName(cve)).String()
}

func getDeferralRequest(expiry *protocompat.Timestamp) *storage.DeferralRequest {
	if expiry == nil {
		re := &storage.RequestExpiry{}
		// Expiry is a OneOf type, and the OneOf wrapper types should be
		// filled with valid data. Reflection-based proto encoding would
		// panic on a non-nil *storage.RequestExpiry_ExpiresOn wrapping
		// a nil timestamp object.
		re.ClearExpiry()
		re.SetExpiryType(storage.RequestExpiry_TIME)
		dr := &storage.DeferralRequest{}
		dr.SetExpiry(re)
		return dr
	}
	re := &storage.RequestExpiry{}
	re.SetExpiryType(storage.RequestExpiry_TIME)
	re.SetExpiresOn(proto.ValueOrDefault(expiry))
	dr := &storage.DeferralRequest{}
	dr.SetExpiry(re)
	return dr
}
