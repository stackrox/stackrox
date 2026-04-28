package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/storagetov2"
	"github.com/stackrox/rox/central/views/vmcve"
	componentDS "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore"
	cveDS "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore"
	scanDS "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore"
	vmDS "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize = 100
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.VirtualMachine)): {
			v2.VirtualMachineV2Service_ListVMs_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEs_FullMethodName,
			v2.VirtualMachineV2Service_GetVMDashboardCounts_FullMethodName,
			v2.VirtualMachineV2Service_GetVM_FullMethodName,
			v2.VirtualMachineV2Service_GetVMVulnSummary_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEsByVM_FullMethodName,
			v2.VirtualMachineV2Service_GetVMCVEComponents_FullMethodName,
			v2.VirtualMachineV2Service_ListVMComponents_FullMethodName,
			v2.VirtualMachineV2Service_GetVMCVEDetail_FullMethodName,
			v2.VirtualMachineV2Service_ListVMCVEAffectedVMs_FullMethodName,
		},
	})
)

type serviceImpl struct {
	v2.UnimplementedVirtualMachineV2ServiceServer

	vmDS        vmDS.DataStore
	cveDS       cveDS.DataStore
	componentDS componentDS.DataStore
	scanDS      scanDS.DataStore
	cveView     vmcve.CveView
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v2.RegisterVirtualMachineV2ServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v2.RegisterVirtualMachineV2ServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// ListVMs returns a paginated list of VMs with severity counts.
func (s *serviceImpl) ListVMs(ctx context.Context, request *v2.ListVMsRequest) (*v2.ListVMsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.vmDS.CountVirtualMachines(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	searchResults, err := s.vmDS.Search(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	vmIDs := make([]string, 0, len(searchResults))
	for _, r := range searchResults {
		vmIDs = append(vmIDs, r.ID)
	}

	if len(vmIDs) == 0 {
		return &v2.ListVMsResponse{TotalCount: int32(totalCount)}, nil
	}

	// Fetch full VM objects for the page.
	vms, _, err := s.vmDS.GetManyVirtualMachines(ctx, vmIDs)
	if err != nil {
		return nil, err
	}

	// Fetch per-VM CVE severity counts via SQL GROUP BY.
	vmFilter := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, vmIDs...).ProtoQuery()
	severityRows, err := s.cveView.CountBySeverityPerVM(ctx, vmFilter)
	if err != nil {
		return nil, err
	}
	severityByVM := make(map[string]*v2.VulnCountBySeverity, len(severityRows))
	for _, row := range severityRows {
		severityByVM[row.GetVMID()] = storagetov2.SeverityCountsToProto(row.GetSeverityCounts())
	}

	// Batch fetch component scan counts. The component Notes field has no dedicated
	// column (no search tag), so SQL-level filtering of scanned vs unscanned is not
	// possible and in-memory aggregation is used.
	componentCountsByVM, err := s.batchComponentScanCounts(ctx, vmIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMListItem, 0, len(vms))
	for _, vm := range vms {
		item := storagetov2.VirtualMachineV2ToListItem(vm)
		if counts, ok := severityByVM[vm.GetId()]; ok {
			item.CveSeverityCounts = counts
		} else {
			item.CveSeverityCounts = &v2.VulnCountBySeverity{}
		}
		item.ComponentScanCount = componentCountsByVM[vm.GetId()]
		items = append(items, item)
	}

	return &v2.ListVMsResponse{
		Vms:        items,
		TotalCount: int32(totalCount),
	}, nil
}

// ListVMCVEs returns a paginated list of CVEs across all VMs.
func (s *serviceImpl) ListVMCVEs(ctx context.Context, request *v2.ListVMCVEsRequest) (*v2.ListVMCVEsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.cveView.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	cves, err := s.cveView.Get(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	// Get total VM count for the response.
	totalVMs, err := s.vmDS.CountVirtualMachines(ctx, search.EmptyQuery())
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMCVEListItem, 0, len(cves))
	for _, cve := range cves {
		item := &v2.VMCVEListItem{
			Cve:              cve.GetCVE(),
			VmSeverityCounts: storagetov2.SeverityCountsToProto(cve.GetVMsBySeverity()),
			TopCvss:          cve.GetTopCVSS(),
			AffectedVmCount:  int32(cve.GetAffectedVMCount()),
			TotalVmCount:     int32(totalVMs),
			EpssProbability:  cve.GetEPSSProbability(),
			PublishedOn:      protocompat.ConvertTimeToTimestampOrNil(cve.GetPublishDate()),
		}
		items = append(items, item)
	}

	return &v2.ListVMCVEsResponse{
		Cves:       items,
		TotalCount: int32(totalCount),
	}, nil
}

// GetVMDashboardCounts returns the total VM and CVE counts for the dashboard.
func (s *serviceImpl) GetVMDashboardCounts(ctx context.Context, request *v2.VMDashboardCountsRequest) (*v2.VMDashboardCountsResponse, error) {
	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}

	vmCount, err := s.vmDS.CountVirtualMachines(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	cveCount, err := s.cveView.Count(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	return &v2.VMDashboardCountsResponse{
		VmCount:  int32(vmCount),
		CveCount: int32(cveCount),
	}, nil
}

// batchComponentScanCounts fetches all components for the given VM IDs in one query
// and counts total vs scanned per VM in memory. The component Notes field is not
// search-indexed, so SQL-level aggregation of scanned vs unscanned is not possible.
func (s *serviceImpl) batchComponentScanCounts(ctx context.Context, vmIDs []string) (map[string]*v2.ComponentScanCount, error) {
	result := make(map[string]*v2.ComponentScanCount, len(vmIDs))
	for _, id := range vmIDs {
		result[id] = &v2.ComponentScanCount{}
	}
	if len(vmIDs) == 0 {
		return result, nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, vmIDs...).ProtoQuery()
	components, err := s.componentDS.SearchRawVMComponents(ctx, q)
	if err != nil {
		return nil, err
	}

	// Components link to scans, not VMs directly. We need to resolve the VM ID
	// via the CVE table's vm_v2_id FK. Instead, count components per scan and
	// map scans to VMs.
	// However, SearchRawVMComponents with a VirtualMachineID filter already
	// joins through the scan table, so we get the right components.
	// We need the scan-to-VM mapping to bucket by VM.
	scanIDs := make(map[string]struct{})
	compsByScan := make(map[string][]*storage.VirtualMachineComponentV2)
	for _, comp := range components {
		scanIDs[comp.GetVmScanId()] = struct{}{}
		compsByScan[comp.GetVmScanId()] = append(compsByScan[comp.GetVmScanId()], comp)
	}

	scanIDList := make([]string, 0, len(scanIDs))
	for id := range scanIDs {
		scanIDList = append(scanIDList, id)
	}
	scans, err := s.scanDS.GetBatch(ctx, scanIDList)
	if err != nil {
		return nil, err
	}

	scanToVM := make(map[string]string, len(scans))
	for _, scan := range scans {
		scanToVM[scan.GetId()] = scan.GetVmV2Id()
	}

	for scanID, comps := range compsByScan {
		vmID, ok := scanToVM[scanID]
		if !ok {
			continue
		}
		counts, ok := result[vmID]
		if !ok {
			continue
		}
		for _, comp := range comps {
			counts.Total++
			scanned := true
			for _, n := range comp.GetNotes() {
				if n == storage.VirtualMachineComponentV2_UNSCANNED {
					scanned = false
					break
				}
			}
			if scanned {
				counts.Scanned++
			}
		}
	}

	return result, nil
}

// GetVMVulnSummary returns vulnerability severity counts for a single VM.
func (s *serviceImpl) GetVMVulnSummary(ctx context.Context, request *v2.GetVMVulnSummaryRequest) (*v2.VMVulnSummary, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id must be specified")
	}

	vmQuery := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, request.GetId()).ProtoQuery()
	count, err := s.vmDS.CountVirtualMachines(ctx, vmQuery)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, status.Errorf(codes.NotFound, "virtual machine %q not found", request.GetId())
	}

	vmFilter := vmQuery.CloneVT()
	if request.GetQuery().GetQuery() != "" {
		additionalQuery, err := search.ParseQuery(request.GetQuery().GetQuery())
		if err != nil {
			return nil, errors.Wrap(err, "parsing input query")
		}
		vmFilter = search.ConjunctionQuery(vmFilter, additionalQuery)
	}

	severityCounts, err := s.cveView.CountBySeverity(ctx, vmFilter)
	if err != nil {
		return nil, err
	}

	proto := storagetov2.SeverityCountsToProto(severityCounts)
	fixable, notFixable := countFixability(proto)

	return &v2.VMVulnSummary{
		SeverityCounts:  proto,
		FixableCount:    fixable,
		NotFixableCount: notFixable,
	}, nil
}

// ListVMCVEsByVM returns a paginated list of CVEs affecting a specific VM.
func (s *serviceImpl) ListVMCVEsByVM(ctx context.Context, request *v2.ListVMCVEsByVMRequest) (*v2.ListVMCVEsByVMResponse, error) {
	if request.GetVmId() == "" {
		return nil, status.Error(codes.InvalidArgument, "vm_id must be specified")
	}

	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	searchQuery = search.ConjunctionQuery(
		searchQuery,
		search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, request.GetVmId()).ProtoQuery(),
	)
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.cveDS.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	cves, err := s.cveDS.SearchRawVMCVEs(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMCVERow, 0, len(cves))
	for _, cve := range cves {
		items = append(items, storagetov2.VirtualMachineCVEV2ToRow(cve))
	}

	return &v2.ListVMCVEsByVMResponse{
		Cves:       items,
		TotalCount: int32(totalCount),
	}, nil
}

// GetVMCVEComponents returns components affected by a specific CVE on a specific VM.
func (s *serviceImpl) GetVMCVEComponents(ctx context.Context, request *v2.GetVMCVEComponentsRequest) (*v2.GetVMCVEComponentsResponse, error) {
	if request.GetVmId() == "" || request.GetCveId() == "" {
		return nil, status.Error(codes.InvalidArgument, "vm_id and cve_id must be specified")
	}

	q := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, request.GetVmId()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.CVE, request.GetCveId()).ProtoQuery(),
	)

	components, err := s.cveView.GetCVEComponents(ctx, q)
	if err != nil {
		return nil, err
	}

	rows := make([]*v2.VMCVEComponentRow, 0, len(components))
	for _, comp := range components {
		var advisory *v2.Advisory
		if comp.GetAdvisoryName() != "" {
			advisory = &v2.Advisory{
				Name: comp.GetAdvisoryName(),
				Link: comp.GetAdvisoryLink(),
			}
		}
		rows = append(rows, &v2.VMCVEComponentRow{
			ComponentName:    comp.GetComponentName(),
			ComponentVersion: comp.GetComponentVersion(),
			Source:           v2.SourceType(comp.GetComponentSource()),
			FixedBy:          comp.GetFixedBy(),
			Advisory:         advisory,
		})
	}

	return &v2.GetVMCVEComponentsResponse{
		Components: rows,
	}, nil
}

// ListVMComponents returns a paginated list of components for a specific VM.
func (s *serviceImpl) ListVMComponents(ctx context.Context, request *v2.ListVMComponentsRequest) (*v2.ListVMComponentsResponse, error) {
	if request.GetVmId() == "" {
		return nil, status.Error(codes.InvalidArgument, "vm_id must be specified")
	}

	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	searchQuery = search.ConjunctionQuery(
		searchQuery,
		search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, request.GetVmId()).ProtoQuery(),
	)
	paginated.FillPaginationV2(searchQuery, request.GetQuery().GetPagination(), defaultPageSize)

	countQuery := searchQuery.CloneVT()
	countQuery.Pagination = nil
	totalCount, err := s.componentDS.Count(ctx, countQuery)
	if err != nil {
		return nil, err
	}

	components, err := s.componentDS.SearchRawVMComponents(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	items := make([]*v2.VMComponentRow, 0, len(components))
	for _, comp := range components {
		items = append(items, storagetov2.VirtualMachineComponentV2ToRow(comp))
	}

	return &v2.ListVMComponentsResponse{
		Components: items,
		TotalCount: int32(totalCount),
	}, nil
}

// countFixability sums fixable and not-fixable counts across all severity levels.
func countFixability(counts *v2.VulnCountBySeverity) (fixable, notFixable int32) {
	for _, sev := range []func() *v2.VulnFixableCount{
		counts.GetCritical,
		counts.GetImportant,
		counts.GetModerate,
		counts.GetLow,
		counts.GetUnknown,
	} {
		c := sev()
		fixable += c.GetFixable()
		notFixable += c.GetTotal() - c.GetFixable()
	}
	return
}

// GetVMCVEDetail returns detailed information about a specific CVE across all VMs.
func (s *serviceImpl) GetVMCVEDetail(ctx context.Context, request *v2.GetVMCVEDetailRequest) (*v2.VMCVEDetail, error) {
	if request.GetCveId() == "" {
		return nil, status.Error(codes.InvalidArgument, "cve_id must be specified")
	}

	// Look up by CVE identifier (e.g. "CVE-2024-1234"), not by internal UUID.
	cveFilter := search.NewQueryBuilder().AddExactMatches(search.CVE, request.GetCveId()).ProtoQuery()
	cves, err := s.cveDS.SearchRawVMCVEs(ctx, cveFilter)
	if err != nil {
		return nil, err
	}
	if len(cves) == 0 {
		return nil, status.Errorf(codes.NotFound, "CVE %q not found", request.GetCveId())
	}
	cve := cves[0]
	severityCounts, err := s.cveView.CountBySeverity(ctx, cveFilter)
	if err != nil {
		return nil, err
	}

	// Get affected VM count.
	affectedVMIDs, err := s.cveView.GetVMIDs(ctx, cveFilter.CloneVT())
	if err != nil {
		return nil, err
	}

	totalVMs, err := s.vmDS.CountVirtualMachines(ctx, search.EmptyQuery())
	if err != nil {
		return nil, err
	}

	// Count distinct guest OSes among affected VMs.
	affectedGuestOSCount := 0
	if len(affectedVMIDs) > 0 {
		affectedVMs, _, err := s.vmDS.GetManyVirtualMachines(ctx, affectedVMIDs)
		if err != nil {
			return nil, err
		}
		guestOSSet := make(map[string]struct{})
		for _, vm := range affectedVMs {
			if os := vm.GetGuestOs(); os != "" {
				guestOSSet[os] = struct{}{}
			}
		}
		affectedGuestOSCount = len(guestOSSet)
	}

	return &v2.VMCVEDetail{
		Cve:                  cve.GetCveBaseInfo().GetCve(),
		Summary:              cve.GetCveBaseInfo().GetSummary(),
		Link:                 cve.GetCveBaseInfo().GetLink(),
		EpssProbability:      cve.GetEpssProbability(),
		PublishedOn:          cve.GetCveBaseInfo().GetPublishedOn(),
		FirstDiscovered:      cve.GetCveBaseInfo().GetCreatedAt(),
		AffectedVmCount:      int32(len(affectedVMIDs)),
		TotalVmCount:         int32(totalVMs),
		AffectedGuestOsCount: int32(affectedGuestOSCount),
		VmSeverityCounts:     storagetov2.SeverityCountsToProto(severityCounts),
		TopCvss:              cve.GetPreferredCvss(),
	}, nil
}

// ListVMCVEAffectedVMs returns VMs affected by a specific CVE.
// TODO(ROX-34181): Replace with a SQL view to enable proper pagination.
func (s *serviceImpl) ListVMCVEAffectedVMs(ctx context.Context, request *v2.ListVMCVEAffectedVMsRequest) (*v2.ListVMCVEAffectedVMsResponse, error) {
	if request.GetCveId() == "" {
		return nil, status.Error(codes.InvalidArgument, "cve_id must be specified")
	}

	searchQuery, err := search.ParseQuery(request.GetQuery().GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(err, "parsing input query")
	}
	searchQuery = search.ConjunctionQuery(
		searchQuery,
		search.NewQueryBuilder().AddExactMatches(search.CVE, request.GetCveId()).ProtoQuery(),
	)

	// Get all CVE records matching this CVE identifier to find affected VMs.
	cves, err := s.cveDS.SearchRawVMCVEs(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	// Build per-VM aggregation: for each VM, pick the highest severity CVE record.
	type vmCVEInfo struct {
		severity       v2.VulnerabilitySeverity
		isFixable      bool
		cvss           float32
		componentCount int
	}
	vmMap := make(map[string]*vmCVEInfo)
	for _, cve := range cves {
		info, ok := vmMap[cve.GetVmV2Id()]
		if !ok {
			info = &vmCVEInfo{}
			vmMap[cve.GetVmV2Id()] = info
		}
		info.componentCount++
		severity := v2.VulnerabilitySeverity(cve.GetSeverity())
		if severity > info.severity {
			info.severity = severity
			info.cvss = cve.GetPreferredCvss()
		}
		if cve.GetIsFixable() {
			info.isFixable = true
		}
	}

	vmIDs := make([]string, 0, len(vmMap))
	for id := range vmMap {
		vmIDs = append(vmIDs, id)
	}

	vms, _, err := s.vmDS.GetManyVirtualMachines(ctx, vmIDs)
	if err != nil {
		return nil, err
	}

	rows := make([]*v2.VMCVEAffectedVMRow, 0, len(vms))
	for _, vm := range vms {
		info := vmMap[vm.GetId()]
		rows = append(rows, &v2.VMCVEAffectedVMRow{
			VmId:                   vm.GetId(),
			VmName:                 vm.GetName(),
			Severity:               info.severity,
			IsFixable:              info.isFixable,
			Cvss:                   info.cvss,
			GuestOs:                vm.GetGuestOs(),
			AffectedComponentCount: int32(info.componentCount),
		})
	}

	return &v2.ListVMCVEAffectedVMsResponse{
		Vms:        rows,
		TotalCount: int32(len(rows)),
	}, nil
}

// GetVM returns detailed information about a single VM.
func (s *serviceImpl) GetVM(ctx context.Context, request *v2.GetVMRequest) (*v2.VMDetail, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id must be specified")
	}

	vm, exists, err := s.vmDS.GetVirtualMachine(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, status.Errorf(codes.NotFound, "virtual machine %q not found", request.GetId())
	}

	detail := storagetov2.VirtualMachineV2ToDetail(vm)

	// Get the latest scan for this VM. Scan IDs are UUIDv7 (time-sortable),
	// so sorting by the primary key is equivalent to sorting by time and avoids
	// a separate index scan.
	scanQuery := search.NewQueryBuilder().AddExactMatches(search.VirtualMachineID, request.GetId()).ProtoQuery()
	scanQuery.Pagination = &v1.QueryPagination{
		Limit: 1,
		SortOptions: []*v1.QuerySortOption{
			{Field: search.VirtualMachineScanID.String(), Reversed: true},
		},
	}
	scans, err := s.scanDS.SearchRawVMScans(ctx, scanQuery)
	if err != nil {
		return nil, err
	}
	if len(scans) > 0 {
		scan := scans[0]
		scanNotes := make([]v2.VMScanNote, 0, len(scan.GetNotes()))
		for _, n := range scan.GetNotes() {
			scanNotes = append(scanNotes, storagetov2.ConvertScanNote(n))
		}
		detail.LatestScan = &v2.VMScanInfo{
			ScanId:    scan.GetId(),
			ScanOs:    scan.GetScanOs(),
			ScanTime:  scan.GetScanTime(),
			TopCvss:   scan.GetTopCvss(),
			ScanNotes: scanNotes,
		}
	}

	return detail, nil
}
