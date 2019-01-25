export const resourceTypes = {
    NAMESPACES: 'namespaces',
    CLUSTERS: 'clusters',
    NODES: 'nodes',
    DEPLOYMENTS: 'deployments'
};

export const standardTypes = {
    PCI: 'PCI',
    NIST: 'NIST',
    HIPAA: 'HIPAA',
    CIS: 'CIS'
};

export default {
    ...resourceTypes,
    ...standardTypes
};
