import axios from 'services/instance';
import type { ScanComponent } from 'types/scanComponent.proto';
import type { SearchQueryOptions } from 'types/search';
import { buildNestedRawQueryParams } from './ComplianceCommon';

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

export function listVMs({
    sortOption,
    page,
    perPage,
    searchFilter,
}: SearchQueryOptions): Promise<ListVMsResponse> {
    const params = buildNestedRawQueryParams({ page, perPage, sortOption, searchFilter });
    return axios
        .get<ListVMsResponse>(`/v2/virtualmachines/vms?${params}`)
        .then((response) => response.data);
}
