import axios from './instance';
import { Empty } from './types';

export type ComplianceStandardScope = 'UNSET' | 'CLUSTER' | 'NAMESPACE' | 'DEPLOYMENT' | 'NODE';

export type ComplianceStandardMetadata = {
    id: string;
    name: string;
    description: string;
    numImplementedChecks: number; // int32
    scopes: ComplianceStandardScope[];
    dynamic: boolean;
    hideScanResults?: boolean; // TODO optional until backend implements new property
};

export function fetchComplianceStandards(): Promise<ComplianceStandardMetadata[]> {
    return axios
        .get<{ standards: ComplianceStandardMetadata[] }>('/v1/compliance/standards')
        .then((response) => {
            return response.data?.standards ?? [];
        });
}

export function patchComplianceStandard(id: string, hideScanResults: boolean): Promise<Empty> {
    return axios
        .patch<Empty>(`/v1/compliance/standard/${id}`, { hideScanResults })
        .then((response) => response.data);
}
