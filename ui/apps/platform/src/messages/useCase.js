import useCaseTypes from 'constants/useCaseTypes';

const legacyPageLabels = {
    [useCaseTypes.CLUSTERS]: 'Clusters',
    [useCaseTypes.RISK]: 'Risk',
    [useCaseTypes.VIOLATIONS]: 'Violations',
    [useCaseTypes.POLICIES]: 'System Policies',
    [useCaseTypes.NETWORK_GRAPH]: 'Network Graph',
    [useCaseTypes.USER]: 'User Profile',
    [useCaseTypes.ACCESS_CONTROL]: 'Access Control',
};

const useCaseLabels = {
    [useCaseTypes.CONFIG_MANAGEMENT]: 'Configuration Management',
    [useCaseTypes.VULN_MANAGEMENT]: 'Vulnerability Management',
    [useCaseTypes.COMPLIANCE]: 'Compliance',
    COMPLIANCE: 'Compliance',
    RISK: 'Risk',
    SECRET: 'Secret',
    POLICY: 'Policy',
    SERVICE_ACCOUNT: 'Service Account',
    DEPLOYMENT: 'Deployment',
    ...legacyPageLabels,
};

export const useCaseShortLabels = {
    [useCaseTypes.CONFIG_MANAGEMENT]: 'CM',
    [useCaseTypes.VULN_MANAGEMENT]: 'VM',
};

export default useCaseLabels;
