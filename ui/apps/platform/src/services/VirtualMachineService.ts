import qs from 'qs';

import axios from 'services/instance';
import type { VulnerabilitySeverity } from 'types/cve.proto';
import type { ScanComponent, SourceType } from 'types/scanComponent.proto';
import type { Advisory } from 'types/vulnerability.proto';
import type { SearchFilter, SearchQueryOptions } from 'types/search';
import {
    applyRegexSearchModifiers,
    buildNestedRawQueryParams,
    getRequestQueryStringForSearchFilter,
} from 'utils/searchUtils';

// Legacy API (v2/virtualmachines)

type VirtualMachineState = 'UNKNOWN' | 'STOPPED' | 'RUNNING';

type VirtualMachineScan = {
    scanTime: string; // ISO 8601 date string
    operatingSystem: string;
    components: ScanComponent[];
    notes: VirtualMachineScanNote[];
};

type VirtualMachineScanNote = 'UNSET' | 'OS_UNKNOWN' | 'OS_UNSUPPORTED';

export type VirtualMachine = {
    id: string;
    namespace: string;
    name: string;
    clusterId: string;
    clusterName: string;
    facts?: Record<string, string>;
    scan?: VirtualMachineScan;
    lastUpdated: string; // ISO 8601 date string
    vsockCid: number;
    state: VirtualMachineState;
};

export type ListVirtualMachinesResponse = {
    virtualMachines: VirtualMachine[];
    totalCount: number;
};

export function listVirtualMachines({
    sortOption,
    page,
    perPage,
    searchFilter,
}: SearchQueryOptions): Promise<ListVirtualMachinesResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
    return axios
        .get<ListVirtualMachinesResponse>(`/v2/virtualmachines?${params}`)
        .then((response) => response.data);
}

export function getVirtualMachine(id: string): Promise<VirtualMachine> {
    return axios.get<VirtualMachine>(`/v2/virtualmachines/${id}`).then((response) => response.data);
}

// Enhanced API (v2/virtualmachines/vms)
// behind ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL feature flag

type VulnFixableCount = {
    total: number;
    fixable: number;
};

export type VulnCountBySeverity = {
    critical: VulnFixableCount;
    important: VulnFixableCount;
    moderate: VulnFixableCount;
    low: VulnFixableCount;
    unknown: VulnFixableCount;
};

export type ComponentScanCount = {
    scanned: number;
    total: number;
};

export type VMListItem = {
    id: string;
    name: string;
    namespace: string;
    clusterId: string;
    clusterName: string;
    guestOs: string;
    state: VirtualMachineState;
    scanTime: string; // ISO 8601 date string
    lastUpdated: string; // ISO 8601 date string
    cveSeverityCounts: VulnCountBySeverity;
    componentScanCount: ComponentScanCount;
};

export type ListVMsResponse = {
    vms: VMListItem[];
    totalCount: number;
};

export type VMCVEListItem = {
    cve: string;
    vmSeverityCounts: VulnCountBySeverity;
    topCvss: number;
    cvssVersion: string;
    affectedVmCount: number;
    totalVmCount: number;
    epssProbability: number;
    publishedOn: string; // ISO 8601 date string
};

export type ListVMCVEsResponse = {
    cves: VMCVEListItem[];
    totalCount: number;
};

export type VMCVEDetail = {
    cve: string;
    summary: string;
    link: string;
    epssProbability: number;
    publishedOn: string;
    firstDiscovered: string;
    affectedVmCount: number;
    totalVmCount: number;
    affectedGuestOsCount: number;
    vmSeverityCounts: VulnCountBySeverity;
    topCvss: number;
};

export type VMCVEAffectedVMRow = {
    vmId: string;
    vmName: string;
    severity: VulnerabilitySeverity;
    isFixable: boolean;
    cvss: number;
    guestOs: string;
    affectedComponentCount: number;
};

export type ListVMCVEAffectedVMsResponse = {
    vms: VMCVEAffectedVMRow[];
    totalCount: number;
};

export type VMCVEComponentRow = {
    componentName: string;
    componentVersion: string;
    source: SourceType;
    fixedBy: string;
    advisory: Advisory | null;
};

export type GetVMCVEComponentsResponse = {
    components: VMCVEComponentRow[];
};

export type VMCVEByVMRow = {
    cve: string;
    severity: VulnerabilitySeverity;
    isFixable: boolean;
    cvss: number;
    nvdCvss: number;
    epssProbability: number;
    affectedComponentCount: number;
    publishedOn?: string;
    summary: string;
    link: string;
    advisory: Advisory | null;
};

export type ListVMCVEsByVMResponse = {
    cves: VMCVEByVMRow[];
    totalCount: number;
};

export type VMComponentScanStatus =
    | 'NOT_SCANNED'
    | 'SCAN_PENDING'
    | 'CPE_MISSING'
    | 'REPO_UNKNOWN'
    | 'SCANNED';

export type VMComponentRow = {
    id: string;
    name: string;
    version: string;
    source: SourceType;
    scanStatus: VMComponentScanStatus;
    lastScanned?: string;
    cveCount: number;
};

export type ListVMComponentsResponse = {
    components: VMComponentRow[];
    totalCount: number;
};

export type VirtualMachineV2State = 'VM_STATE_UNKNOWN' | 'VM_STATE_STOPPED' | 'VM_STATE_RUNNING';

export type VMScanNote =
    | 'VM_SCAN_NOTE_UNSET'
    | 'VM_SCAN_NOTE_OS_UNKNOWN'
    | 'VM_SCAN_NOTE_OS_UNSUPPORTED';

export type VMNote =
    | 'VM_NOTE_MISSING_METADATA'
    | 'VM_NOTE_MISSING_SCAN_DATA'
    | 'VM_NOTE_MISSING_SIGNATURE'
    | 'VM_NOTE_MISSING_SIGNATURE_VERIFICATION_DATA'
    | 'VM_NOTE_MISSING_SCANNER'
    | 'VM_NOTE_SCAN_FAILED';

export type AgentStatus = 'AGENT_STATUS_UNKNOWN' | 'AGENT_STATUS_ACTIVE';

export type VMScanInfo = {
    scanId: string;
    scanOs: string;
    scanTime?: string;
    topCvss: number;
    scanNotes: VMScanNote[];
};

export type VMDetail = {
    id: string;
    name: string;
    namespace: string;
    clusterId: string;
    clusterName: string;
    guestOs: string;
    state: VirtualMachineV2State;
    lastUpdated?: string;
    facts: Record<string, string>;
    annotations: Record<string, string>;
    labels: Record<string, string>;
    vsockCid: number;
    notes: VMNote[];
    latestScan?: VMScanInfo;
    agentStatus: AgentStatus;
};

export function listVMs({
    sortOption,
    page,
    perPage,
    searchFilter,
}: SearchQueryOptions): Promise<ListVMsResponse> {
    const params = buildNestedRawQueryParams({
        page,
        perPage,
        sortOption,
        searchFilter: applyRegexSearchModifiers(searchFilter ?? {}),
    });
    return axios
        .get<ListVMsResponse>(`/v2/virtualmachines/vms?${params}`)
        .then((response) => response.data);
}

export function listVMCVEs({
    searchFilter,
    page,
    perPage,
}: SearchQueryOptions): Promise<ListVMCVEsResponse> {
    const params = buildNestedRawQueryParams({
        page,
        perPage,
        searchFilter: applyRegexSearchModifiers(searchFilter ?? {}),
    });
    return axios
        .get<ListVMCVEsResponse>(`/v2/virtualmachines/cves?${params}`)
        .then((response) => response.data);
}

export function getVMCVEDetail(cveId: string, searchFilter: SearchFilter): Promise<VMCVEDetail> {
    // Might consider updating buildNestedRawQueryParams to handle this case (no pagination).
    const params = qs.stringify(
        {
            query: {
                query: getRequestQueryStringForSearchFilter(
                    applyRegexSearchModifiers(searchFilter)
                ),
            },
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get<VMCVEDetail>(`/v2/virtualmachines/cves/${cveId}?${params}`)
        .then((response) => response.data);
}

export function listVMCVEAffectedVMs(
    cveId: string,
    { searchFilter, sortOption, page, perPage }: SearchQueryOptions
): Promise<ListVMCVEAffectedVMsResponse> {
    const params = buildNestedRawQueryParams({
        page,
        perPage,
        sortOption,
        searchFilter: applyRegexSearchModifiers(searchFilter ?? {}),
    });
    return axios
        .get<ListVMCVEAffectedVMsResponse>(`/v2/virtualmachines/cves/${cveId}/vms?${params}`)
        .then((response) => response.data);
}

export function getVMCVEComponents(
    vmId: string,
    cveId: string
): Promise<GetVMCVEComponentsResponse> {
    return axios
        .get<GetVMCVEComponentsResponse>(`/v2/virtualmachines/${vmId}/cves/${cveId}/components`)
        .then((response) => response.data);
}

export function listVMCVEsByVM(
    vmId: string,
    { searchFilter, page, perPage, sortOption }: SearchQueryOptions
): Promise<ListVMCVEsByVMResponse> {
    const params = buildNestedRawQueryParams({
        page,
        perPage,
        searchFilter: applyRegexSearchModifiers(searchFilter ?? {}),
        sortOption,
    });
    return axios
        .get<ListVMCVEsByVMResponse>(`/v2/virtualmachines/${vmId}/cves?${params}`)
        .then((response) => response.data);
}

export function listVMComponents(
    vmId: string,
    { searchFilter, page, perPage, sortOption }: SearchQueryOptions
): Promise<ListVMComponentsResponse> {
    const params = buildNestedRawQueryParams({
        page,
        perPage,
        searchFilter: applyRegexSearchModifiers(searchFilter ?? {}),
        sortOption,
    });
    return axios
        .get<ListVMComponentsResponse>(`/v2/virtualmachines/${vmId}/components?${params}`)
        .then((response) => response.data);
}

export function getVM(vmId: string): Promise<VMDetail> {
    return axios.get<VMDetail>(`/v2/virtualmachines/${vmId}`).then((response) => response.data);
}
