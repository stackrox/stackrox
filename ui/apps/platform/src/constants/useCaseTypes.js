const useCaseTypes = {
    CONFIG_MANAGEMENT: 'configmanagement',
    VULN_MANAGEMENT: 'vulnerability-management',
    COMPLIANCE: 'compliance',
    CLUSTERS: 'clusters',
    RISK: 'risk',
    SECRET: 'secrets',
    POLICY: 'policy',
    SERVICE_ACCOUNT: 'serviceaccounts',
    DEPLOYMENT: 'risk',
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
