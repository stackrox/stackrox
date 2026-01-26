import axios from 'services/instance';
import type { ScanComponent } from 'types/scanComponent.proto';
import type { SearchQueryOptions } from 'types/search';
import { buildNestedRawQueryParams } from './ComplianceCommon';

type VirtualMachineState = 'UNKNOWN' | 'STOPPED' | 'RUNNING';

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

type VirtualMachineScan = {
    scanTime: string; // ISO 8601 date string
    operatingSystem: string;
    components: ScanComponent[];
    notes: VirtualMachineScanNote[];
};

type VirtualMachineScanNote = 'UNSET' | 'OS_UNKNOWN' | 'OS_UNSUPPORTED';

export type ListVirtualMachinesResponse = {
    virtualMachines: VirtualMachine[];
    totalCount: number;
};

/**
 * fetches the list of virtual machines
 */
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

/**
 * fetches a single virtual machine
 */
export function getVirtualMachine(id: string): Promise<VirtualMachine> {
    return axios.get<VirtualMachine>(`/v2/virtualmachines/${id}`).then((response) => response.data);
}
