export const resourceTypes = {
    NAMESPACE: 'NAMESPACE',
    CLUSTER: 'CLUSTER',
    NODE: 'NODE',
    DEPLOYMENT: 'DEPLOYMENT',
    NETWORK_POLICY: 'NETWORK_POLICY'
};

export const standardEntityTypes = {
    CONTROL: 'control',
    GROUP: 'group'
};

export const standardTypes = {
    PCI_DSS_3_2: 'PCI_DSS_3_2',
    NIST_800_190: 'NIST_800_190',
    HIPAA_164: 'HIPAA_164',
    CIS_KUBERENETES_V1_2_0: 'CIS_Kubernetes_v1_2_0',
    CIS_DOCKER_V1_1_0: 'CIS_Docker_v1_1_0'
};

export default {
    ...resourceTypes,
    ...standardTypes
};
