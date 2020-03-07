import useCaseTypes from 'constants/useCaseTypes';

const useCaseLabels = {
    [useCaseTypes.CONFIG_MANAGEMENT]: 'Configuration Management',
    [useCaseTypes.VULN_MANAGEMENT]: 'Vulnerability Management',
    COMPLIANCE: 'Compliance',
    RISK: 'Risk',
    SECRET: 'Secret',
    POLICY: 'Policy',
    SERVICE_ACCOUNT: 'Service Account',
    DEPLOYMENT: 'Deployment'
};

export const useCaseShortLabels = {
    [useCaseTypes.CONFIG_MANAGEMENT]: 'CM',
    [useCaseTypes.VULN_MANAGEMENT]: 'VM'
};

export default useCaseLabels;
