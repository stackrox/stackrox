export const resourceTypes = {
    NAMESPACES: 'namespaces',
    CLUSTERS: 'clusters',
    NODES: 'nodes',
    DEPLOYMENTS: 'deployments'
};

export const standardEntityTypes = {
    CONTROL: 'control',
    GROUP: 'group'
};

export const standardTypes = {
    PCI_DSS_3_2: 'PCI DSS 3.2',
    NIST_800_190: 'NIST 800-190',
    HIPAA_164: 'HIPAA 164',
    CIS_KUBERENETES_V1_2_0: 'CIS Kubernetes v1.2.0',
    CIS_DOCKER_V1_1_0: 'CIS Docker v1.1.0'
};

export default {
    ...resourceTypes,
    ...standardTypes
};
