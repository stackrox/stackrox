/*
export type UseCase =
    | 'ACCESS_CONTROL'
    | 'CLUSTERS'
    | 'COMPLIANCE'
    | 'CONFIG_MANAGEMENT'
    | 'DEPLOYMENT'
    | 'POLICY'
    | 'RISK'
    | 'SECRET'
    | 'SERVICE_ACCOUNT'
    | 'VULN_MANAGEMENT';
*/

// TODO compare to routePaths
const useCaseTypes = {
    ACCESS_CONTROL: 'access-control',
    CONFIG_MANAGEMENT: 'configmanagement',
    VULN_MANAGEMENT: 'vulnerability-management',
    COMPLIANCE: 'compliance',
    CLUSTERS: 'clusters',
    RISK: 'risk',
    SECRET: 'secrets',
    POLICY: 'policy',
    SERVICE_ACCOUNT: 'serviceaccounts',
    DEPLOYMENT: 'risk',
    VIOLATIONS: 'violations',
    POLICIES: 'policies',
    NETWORK_GRAPH: 'network-graph',
    USER: 'user',
};

// TODO: long-term, need to standardize all sections to fully use Workflow State
//   concurrent,
//   Vuln Mgmt fully uses it
//   Risk uses it for search, so it's included in this list
export const newWorkflowCases = [
    useCaseTypes.VULN_MANAGEMENT,
    useCaseTypes.RISK,
    useCaseTypes.CLUSTERS,
];

export default useCaseTypes;
