package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/hashstructure"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store"
	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var log = logging.LoggerForModule()

const (
	vmTable        = pkgSchema.VirtualMachineV2TableName
	scanTable      = pkgSchema.VirtualMachineScanV2TableName
	componentTable = pkgSchema.VirtualMachineComponentV2TableName
	cveTable       = pkgSchema.VirtualMachineCvev2TableName

	getVMStmt   = "SELECT serialized FROM " + vmTable + " WHERE Id = $1"
	getScanStmt = "SELECT serialized FROM " + scanTable + " WHERE VmV2Id = $1"
)

var (
	schema = pkgSchema.VirtualMachineV2Schema
)

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB, keyFence concurrency.KeyFence) store.Store {
	return &storeImpl{
		db:       db,
		keyFence: keyFence,
	}
}

type storeImpl struct {
	db       postgres.DB
	keyFence concurrency.KeyFence
}

// region UpsertVM

func (s *storeImpl) UpsertVM(ctx context.Context, vm *storage.VirtualMachineV2) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "VirtualMachineV2")

	return pgutils.Retry(ctx, func() error {
		return s.upsertVM(ctx, vm)
	})
}

func (s *storeImpl) upsertVM(ctx context.Context, vm *storage.VirtualMachineV2) error {
	// Calculate hash to see if anything changed
	hash, err := hashVM(vm)
	if err != nil {
		return err
	}

	iTime := time.Now()
	vm.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&iTime)

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(vm.GetId())), func() error {
		tx, ctx, err := s.begin(ctx)
		if err != nil {
			return err
		}

		existingVM, err := s.getVirtualMachine(ctx, tx, vm.GetId())
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return errors.Wrapf(rbErr, "rollback after: %v", err)
			}
			return err
		}

		if existingVM != nil && existingVM.GetHash() == hash {
			// Unchanged: timestamp-only update. Preserve the existing hash.
			vm.Hash = hash
			if err := s.updateVMTimestamp(ctx, tx, vm); err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					return errors.Wrapf(rbErr, "rollback after: %v", err)
				}
				return err
			}
			return tx.Commit(ctx)
		}

		// Changed or new: full upsert.
		vm.Hash = hash
		if err := s.insertVM(ctx, tx, vm); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return errors.Wrapf(rbErr, "rollback after: %v", err)
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

func hashVM(vm *storage.VirtualMachineV2) (uint64, error) {
	return hashstructure.Hash(vm, &hashstructure.HashOptions{ZeroNil: true})
}

func (s *storeImpl) insertVM(ctx context.Context, tx *postgres.Tx, vm *storage.VirtualMachineV2) error {
	id := pgutils.NilOrUUID(vm.GetId())
	if id == nil {
		return errors.New("VM ID is empty or not a valid UUID")
	}

	serialized, err := vm.MarshalVT()
	if err != nil {
		return err
	}

	values := []interface{}{
		id,
		vm.GetName(),
		vm.GetNamespace(),
		pgutils.NilOrUUID(vm.GetClusterId()),
		vm.GetClusterName(),
		vm.GetGuestOs(),
		vm.GetState(),
		protocompat.NilOrTime(vm.GetLastUpdated()),
		serialized,
	}

	const stmt = "INSERT INTO " + vmTable +
		" (Id, Name, Namespace, ClusterId, ClusterName, GuestOs, State, LastUpdated, serialized)" +
		" VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)" +
		" ON CONFLICT(Id) DO UPDATE SET" +
		" Id = EXCLUDED.Id, Name = EXCLUDED.Name, Namespace = EXCLUDED.Namespace," +
		" ClusterId = EXCLUDED.ClusterId, ClusterName = EXCLUDED.ClusterName," +
		" GuestOs = EXCLUDED.GuestOs, State = EXCLUDED.State," +
		" LastUpdated = EXCLUDED.LastUpdated, serialized = EXCLUDED.serialized"
	_, err = tx.Exec(ctx, stmt, values...)
	return err
}

func (s *storeImpl) updateVMTimestamp(ctx context.Context, tx *postgres.Tx, vm *storage.VirtualMachineV2) error {
	id := pgutils.NilOrUUID(vm.GetId())
	if id == nil {
		return errors.New("VM ID is empty or not a valid UUID")
	}

	serialized, err := vm.MarshalVT()
	if err != nil {
		return err
	}

	const stmt = "UPDATE " + vmTable + " SET LastUpdated = $2, serialized = $3 WHERE Id = $1"
	_, err = tx.Exec(ctx, stmt, id, protocompat.NilOrTime(vm.GetLastUpdated()), serialized)
	return err
}

func (s *storeImpl) getVirtualMachine(ctx context.Context, tx *postgres.Tx, id string) (*storage.VirtualMachineV2, error) {
	row := tx.QueryRow(ctx, getVMStmt, pgutils.NilOrUUID(id))
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	var vm storage.VirtualMachineV2
	if err := vm.UnmarshalVTUnsafe(data); err != nil {
		return nil, err
	}
	return &vm, nil
}

// endregion UpsertVM

// region UpsertScan

func (s *storeImpl) UpsertScan(ctx context.Context, vmID string, parts common.VMScanParts) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "VirtualMachineScanV2")

	return pgutils.Retry(ctx, func() error {
		return s.upsertScan(ctx, vmID, parts)
	})
}

// scanHashWrapper hashes the original v1 scan components directly, relying on
// @gotags in the proto definitions to ignore derived/store-set fields
// (top_cvss, risk_score, created_at). This matches the image store's pattern.
type scanHashWrapper struct {
	ScanOs     string
	Components []*storage.EmbeddedVirtualMachineScanComponent `hash:"set"`
}

func buildScanHash(parts common.VMScanParts) (uint64, error) {
	return hashstructure.Hash(scanHashWrapper{
		ScanOs:     parts.Scan.GetScanOs(),
		Components: parts.SourceComponents,
	}, &hashstructure.HashOptions{ZeroNil: true})
}

func (s *storeImpl) upsertScan(ctx context.Context, vmID string, parts common.VMScanParts) error {
	if parts.Scan == nil {
		return errors.New("cannot upsert scan: scan is nil")
	}
	iTime := time.Now()

	hash, err := buildScanHash(parts)
	if err != nil {
		return errors.Wrap(err, "computing scan hash")
	}

	log.Infof("VM v2 store: upserting scan for VM %s (scan=%s, hash=%d, %d components, %d CVEs)",
		vmID, parts.Scan.GetId(), hash, len(parts.Components), len(parts.CVEs))

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(vmID)), func() error {
		tx, ctx, err := s.begin(ctx)
		if err != nil {
			return err
		}

		// Touch VM last_updated.
		if err := s.touchVMLastUpdated(ctx, tx, vmID, iTime); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return errors.Wrapf(rbErr, "rollback after: %v", err)
			}
			return err
		}

		// Read existing scan for VM.
		existingScan, err := s.getScanForVM(ctx, tx, vmID)
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return errors.Wrapf(rbErr, "rollback after: %v", err)
			}
			return err
		}

		scanTime := protocompat.ConvertTimeToTimestampOrNil(&iTime)
		parts.Scan.ScanTime = scanTime

		if existingScan != nil && existingScan.GetHash() == hash {
			log.Infof("VM v2 store: scan unchanged for VM %s (existing scan=%s), updating scan time only",
				vmID, existingScan.GetId())
			// Unchanged: scan_time-only update using existing scan's identity.
			existingScan.ScanTime = scanTime
			if err := s.updateScanTime(ctx, tx, existingScan.GetId(), existingScan); err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					return errors.Wrapf(rbErr, "rollback after: %v", err)
				}
				return err
			}
			return tx.Commit(ctx)
		}

		// Changed or new: full replace.
		parts.Scan.Hash = hash

		if existingScan == nil {
			log.Infof("VM v2 store: no existing scan for VM %s, performing full insert", vmID)
		} else {
			log.Infof("VM v2 store: scan changed for VM %s (old hash=%d, new hash=%d), performing full replace",
				vmID, existingScan.GetHash(), hash)
		}

		if err := s.fullScanReplace(ctx, tx, vmID, parts, iTime); err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				return errors.Wrapf(rbErr, "rollback after: %v", err)
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

func (s *storeImpl) touchVMLastUpdated(ctx context.Context, tx *postgres.Tx, vmID string, t time.Time) error {
	// Read the current VM to update its serialized blob with the new timestamp.
	existingVM, err := s.getVirtualMachine(ctx, tx, vmID)
	if err != nil {
		return errors.Wrapf(err, "reading VM %s for timestamp update", vmID)
	}
	if existingVM == nil {
		return errors.Errorf("VM %s not found", vmID)
	}
	existingVM.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&t)
	return s.updateVMTimestamp(ctx, tx, existingVM)
}

func (s *storeImpl) getScanForVM(ctx context.Context, tx *postgres.Tx, vmID string) (*storage.VirtualMachineScanV2, error) {
	row := tx.QueryRow(ctx, getScanStmt, pgutils.NilOrUUID(vmID))
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	var scan storage.VirtualMachineScanV2
	if err := scan.UnmarshalVTUnsafe(data); err != nil {
		return nil, err
	}
	return &scan, nil
}

func (s *storeImpl) updateScanTime(ctx context.Context, tx *postgres.Tx, scanID string, scan *storage.VirtualMachineScanV2) error {
	id := pgutils.NilOrUUID(scanID)
	if id == nil {
		return errors.New("scan ID is empty or not a valid UUID")
	}

	serialized, err := scan.MarshalVT()
	if err != nil {
		return err
	}

	const stmt = "UPDATE " + scanTable + " SET ScanTime = $2, serialized = $3 WHERE Id = $1"
	_, err = tx.Exec(ctx, stmt, id, protocompat.NilOrTime(scan.GetScanTime()), serialized)
	return err
}

func (s *storeImpl) fullScanReplace(ctx context.Context, tx *postgres.Tx, vmID string, parts common.VMScanParts, iTime time.Time) error {
	// Build CVE time map from incoming CVEs.
	log.Infof("VM v2 store: fullScanReplace step 1: building CVE time map for VM %s (%d CVEs)", vmID, len(parts.CVEs))
	cveTimeMap := buildCVETimeMap(parts.CVEs, iTime)

	// Query existing CVE times.
	log.Infof("VM v2 store: fullScanReplace step 2: querying existing CVE times for VM %s", vmID)
	existingTimes, err := s.getExistingCVETimes(ctx, tx, vmID)
	if err != nil {
		return err
	}

	// Merge: keep oldest created_at per CVE.
	log.Infof("VM v2 store: fullScanReplace step 3: merging %d existing CVE times for VM %s", len(existingTimes), vmID)
	for cve, existingTime := range existingTimes {
		if incoming, ok := cveTimeMap[cve]; ok {
			if existingTime.Before(incoming) {
				cveTimeMap[cve] = existingTime
			}
		}
	}

	// Apply merged timestamps to incoming CVE objects.
	log.Infof("VM v2 store: fullScanReplace step 4: applying merged timestamps to %d CVEs for VM %s", len(parts.CVEs), vmID)
	for _, cve := range parts.CVEs {
		cveName := cve.GetCveBaseInfo().GetCve()
		if t, ok := cveTimeMap[cveName]; ok {
			if cve.GetCveBaseInfo() == nil {
				cve.CveBaseInfo = &storage.CVEInfo{}
			}
			cve.CveBaseInfo.CreatedAt = timestamppb.New(t)
		}
	}

	// Delete old scan (cascade deletes components + CVEs).
	log.Infof("VM v2 store: fullScanReplace step 5: deleting old scan for VM %s", vmID)
	if _, err := tx.Exec(ctx, "DELETE FROM "+scanTable+" WHERE VmV2Id = $1", pgutils.NilOrUUID(vmID)); err != nil {
		return errors.Wrap(err, "deleting old scan")
	}

	// Insert new scan row.
	log.Infof("VM v2 store: fullScanReplace step 6: inserting scan %s for VM %s", parts.Scan.GetId(), vmID)
	if err := s.insertScan(ctx, tx, parts.Scan); err != nil {
		return err
	}

	// COPY FROM components (batched).
	log.Infof("VM v2 store: fullScanReplace step 7: COPY %d components for VM %s", len(parts.Components), vmID)
	if err := s.copyFromComponents(ctx, tx, parts.Components); err != nil {
		return err
	}

	// COPY FROM CVEs (batched).
	log.Infof("VM v2 store: fullScanReplace step 8: COPY %d CVEs for VM %s", len(parts.CVEs), vmID)
	return s.copyFromCVEs(ctx, tx, parts.CVEs)
}

func buildCVETimeMap(cves []*storage.VirtualMachineCVEV2, iTime time.Time) map[string]time.Time {
	cveTimeMap := make(map[string]time.Time, len(cves))
	for _, cve := range cves {
		cveName := cve.GetCveBaseInfo().GetCve()
		createdAt := cve.GetCveBaseInfo().GetCreatedAt()
		t := iTime
		if createdAt != nil {
			t = createdAt.AsTime()
		}
		if existing, ok := cveTimeMap[cveName]; ok {
			if t.Before(existing) {
				cveTimeMap[cveName] = t
			}
		} else {
			cveTimeMap[cveName] = t
		}
	}
	return cveTimeMap
}

func (s *storeImpl) getExistingCVETimes(ctx context.Context, tx *postgres.Tx, vmID string) (map[string]time.Time, error) {
	rows, err := tx.Query(ctx, "SELECT cvebaseinfo_cve, cvebaseinfo_createdat FROM "+cveTable+" WHERE vmv2id = $1", pgutils.NilOrUUID(vmID))
	if err != nil {
		return nil, errors.Wrap(err, "querying existing CVE times")
	}
	defer rows.Close()

	result := make(map[string]time.Time)
	for rows.Next() {
		var cveName string
		var createdAt *time.Time
		if err := rows.Scan(&cveName, &createdAt); err != nil {
			return nil, err
		}
		if createdAt != nil {
			if existing, ok := result[cveName]; ok {
				if createdAt.Before(existing) {
					result[cveName] = *createdAt
				}
			} else {
				result[cveName] = *createdAt
			}
		}
	}
	return result, rows.Err()
}

func (s *storeImpl) insertScan(ctx context.Context, tx *postgres.Tx, scan *storage.VirtualMachineScanV2) error {
	id := pgutils.NilOrUUID(scan.GetId())
	if id == nil {
		return errors.New("scan ID is empty or not a valid UUID")
	}
	vmID := pgutils.NilOrUUID(scan.GetVmV2Id())
	if vmID == nil {
		return errors.New("scan VM ID is empty or not a valid UUID")
	}

	serialized, err := scan.MarshalVT()
	if err != nil {
		return err
	}

	values := []interface{}{
		id,
		vmID,
		protocompat.NilOrTime(scan.GetScanTime()),
		scan.GetTopCvss(),
		serialized,
	}

	const stmt = "INSERT INTO " + scanTable +
		" (Id, VmV2Id, ScanTime, TopCvss, serialized)" +
		" VALUES($1, $2, $3, $4, $5)"
	_, err = tx.Exec(ctx, stmt, values...)
	return err
}

func (s *storeImpl) copyFromComponents(ctx context.Context, tx *postgres.Tx, components []*storage.VirtualMachineComponentV2) error {
	if len(components) == 0 {
		return nil
	}

	batchSize := pgSearch.MaxBatchSize
	if len(components) < batchSize {
		batchSize = len(components)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"id",
		"vmscanid",
		"name",
		"version",
		"source",
		"operatingsystem",
		"topcvss",
		"serialized",
	}

	for idx, obj := range components {
		id := pgutils.NilOrUUID(obj.GetId())
		if id == nil {
			return errors.Errorf("component %d ID is empty or not a valid UUID", idx)
		}
		scanID := pgutils.NilOrUUID(obj.GetVmScanId())
		if scanID == nil {
			return errors.Errorf("component %d scan ID is empty or not a valid UUID", idx)
		}

		serialized, err := obj.MarshalVT()
		if err != nil {
			return err
		}

		inputRows = append(inputRows, []interface{}{
			id,
			scanID,
			obj.GetName(),
			obj.GetVersion(),
			obj.GetSource(),
			obj.GetOperatingSystem(),
			obj.GetTopCvss(),
			serialized,
		})

		if (idx+1)%batchSize == 0 || idx == len(components)-1 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{componentTable}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			inputRows = inputRows[:0]
		}
	}

	return nil
}

func (s *storeImpl) copyFromCVEs(ctx context.Context, tx *postgres.Tx, cves []*storage.VirtualMachineCVEV2) error {
	if len(cves) == 0 {
		return nil
	}

	batchSize := pgSearch.MaxBatchSize
	if len(cves) < batchSize {
		batchSize = len(cves)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"id",
		"vmv2id",
		"vmcomponentid",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"cvebaseinfo_epss_epssprobability",
		"preferredcvss",
		"severity",
		"impactscore",
		"nvdcvss",
		"isfixable",
		"fixedby",
		"epssprobability",
		"advisory_name",
		"advisory_link",
		"serialized",
	}

	for idx, obj := range cves {
		id := pgutils.NilOrUUID(obj.GetId())
		if id == nil {
			return errors.Errorf("CVE %d ID is empty or not a valid UUID", idx)
		}
		vmID := pgutils.NilOrUUID(obj.GetVmV2Id())
		if vmID == nil {
			return errors.Errorf("CVE %d VM ID is empty or not a valid UUID", idx)
		}
		compID := pgutils.NilOrUUID(obj.GetVmComponentId())
		if compID == nil {
			return errors.Errorf("CVE %d component ID is empty or not a valid UUID", idx)
		}

		serialized, err := obj.MarshalVT()
		if err != nil {
			return err
		}

		inputRows = append(inputRows, []interface{}{
			id,
			vmID,
			compID,
			obj.GetCveBaseInfo().GetCve(),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
			obj.GetPreferredCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetNvdcvss(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			obj.GetEpssProbability(),
			obj.GetAdvisory().GetName(),
			obj.GetAdvisory().GetLink(),
			serialized,
		})

		if (idx+1)%batchSize == 0 || idx == len(cves)-1 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{cveTable}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			inputRows = inputRows[:0]
		}
	}

	return nil
}

// endregion UpsertScan

// region Delete

func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "VirtualMachineV2")

	return pgutils.Retry(ctx, func() error {
		return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet([]byte(id)), func() error {
			tx, ctx, err := s.begin(ctx)
			if err != nil {
				return err
			}
			// FK cascade handles scan, component, CVE deletion.
			if _, err := tx.Exec(ctx, "DELETE FROM "+vmTable+" WHERE Id = $1", pgutils.NilOrUUID(id)); err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					return errors.Wrapf(rbErr, "rollback after: %v", err)
				}
				return err
			}
			return tx.Commit(ctx)
		})
	})
}

// DeleteMany removes multiple VMs and all associated data.
// NOTE: This does not acquire per-ID keyFence locks for performance reasons,
// matching the pattern used by other stores (e.g., image store). Callers
// should avoid concurrent upserts on the same IDs during batch deletes.
func (s *storeImpl) DeleteMany(ctx context.Context, ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "VirtualMachineV2")

	if len(ids) == 0 {
		return nil
	}

	return pgutils.Retry(ctx, func() error {
		q := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
		return pgSearch.RunDeleteRequestForSchema(ctx, schema, q, s.db)
	})
}

// endregion Delete

// region Read operations

func (s *storeImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "VirtualMachineV2")

	return pgutils.Retry2(ctx, func() (int, error) {
		return pgSearch.RunCountRequestForSchema(ctx, schema, q, s.db)
	})
}

func (s *storeImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Search, "VirtualMachineV2")

	return pgutils.Retry2(ctx, func() ([]search.Result, error) {
		return pgSearch.RunSearchRequestForSchema(ctx, schema, q, s.db)
	})
}

func (s *storeImpl) Get(ctx context.Context, id string) (*storage.VirtualMachineV2, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "VirtualMachineV2")

	return pgutils.Retry3(ctx, func() (*storage.VirtualMachineV2, bool, error) {
		tx, ctx, err := s.begin(ctx)
		if err != nil {
			return nil, false, err
		}
		defer postgres.FinishReadOnlyTransaction(tx)

		vm, err := s.getVirtualMachine(ctx, tx, id)
		if err != nil {
			return nil, false, err
		}
		if vm == nil {
			return nil, false, nil
		}
		return vm, true, nil
	})
}

func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.VirtualMachineV2, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "VirtualMachineV2")

	return pgutils.Retry3(ctx, func() ([]*storage.VirtualMachineV2, []int, error) {
		if len(ids) == 0 {
			return nil, nil, nil
		}

		q := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
		resultsByID := make(map[string]*storage.VirtualMachineV2, len(ids))
		err := pgSearch.RunQueryForSchemaFn[storage.VirtualMachineV2](ctx, schema, q, s.db, func(vm *storage.VirtualMachineV2) error {
			resultsByID[vm.GetId()] = vm
			return nil
		})
		if err != nil {
			return nil, nil, err
		}

		// Preserve input order and track missing indices.
		elems := make([]*storage.VirtualMachineV2, 0, len(resultsByID))
		missingIndices := make([]int, 0, len(ids)-len(resultsByID))
		for i, id := range ids {
			if vm, ok := resultsByID[id]; ok {
				elems = append(elems, vm)
			} else {
				missingIndices = append(missingIndices, i)
			}
		}
		return elems, missingIndices, nil
	})
}

func (s *storeImpl) Walk(ctx context.Context, fn func(vm *storage.VirtualMachineV2) error) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Walk, "VirtualMachineV2")

	return pgSearch.RunCursorQueryForSchemaFn[storage.VirtualMachineV2](ctx, schema, search.EmptyQuery(), s.db, fn)
}

func (s *storeImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(vm *storage.VirtualMachineV2) error) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.WalkByQuery, "VirtualMachineV2")

	return pgSearch.RunCursorQueryForSchemaFn[storage.VirtualMachineV2](ctx, schema, q, s.db, fn)
}

// endregion Read operations

func (s *storeImpl) begin(ctx context.Context) (*postgres.Tx, context.Context, error) {
	return postgres.GetTransaction(ctx, s.db)
}
