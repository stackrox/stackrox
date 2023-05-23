import axios from './instance';
import { Empty } from './types';

const standardsUrl = '/v1/compliance/standards';

export type ComplianceStandardScope = 'UNSET' | 'CLUSTER' | 'NAMESPACE' | 'DEPLOYMENT' | 'NODE';

export type ComplianceStandardMetadata = {
    id: string;
    name: string;
    description: string;
    numImplementedChecks: number; // int32
    scopes: ComplianceStandardScope[];
    dynamic: boolean;
    hidden: boolean;
};

export function fetchComplianceStandards(): Promise<ComplianceStandardMetadata[]> {
    return axios
        .get<{ standards: ComplianceStandardMetadata[] }>(standardsUrl)
        .then((response) => response.data?.standards ?? []);
}

export function patchComplianceStandard(id: string, hidden: boolean): Promise<Empty> {
    return axios
        .patch<Empty>(`${standardsUrl}/${id}`, { hidden })
        .then((response) => response.data);
}
