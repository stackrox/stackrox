import axios from 'services/instance';
import type { ScanComponent } from 'types/scanComponent.proto';

export type VirtualMachine = {
    id: string;
    namespace: string;
    name: string;
    clusterId: string;
    clusterName: string;
    facts: Record<string, string>;
    scan: VirtualMachineScan;
    lastUpdated: string; // ISO 8601 date string
};

type VirtualMachineScan = {
    scannerVersion: string;
    scanTime: string; // ISO 8601 date string
    components: ScanComponent[];
    dataSource: DataSource;
    notes: VirtualMachineScanNote[];
};

type VirtualMachineScanNote =
    | 'UNSET'
    | 'OS_UNAVAILABLE'
    | 'PARTIAL_SCAN_DATA'
    | 'OS_CVES_UNAVAILABLE'
    | 'OS_CVES_STALE'
    | 'LANGUAGE_CVES_UNAVAILABLE'
    | 'CERTIFIED_RHEL_SCAN_UNAVAILABLE';

type DataSource = {
    id: string;
    name: string;
    mirror: string;
};

/**
 * fetches the list of virtual machines
 */
export function listVirtualMachines(): Promise<VirtualMachine[]> {
    return axios.get<VirtualMachine[]>('/v2/virtualmachines').then((response) => response.data);
}

/**
 * fetches a single virtual machine
 */
export function getVirtualMachine(id: string): Promise<VirtualMachine> {
    return axios.get<VirtualMachine>(`/v2/virtualmachines/${id}`).then((response) => response.data);
}
