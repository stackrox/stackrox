import entityTypes from 'constants/entityTypes';

import getEntityName from './getEntityName';

describe('getEntityName', () => {
    it('returns null if data is null', () => {
        const entityType = entityTypes.IMAGE;
        const data = null;

        const entityName = getEntityName(entityType, data);

        expect(entityName).toBe(null);
    });

    it('returns null if data is an empty object', () => {
        const entityType = entityTypes.IMAGE;
        const data = {};

        const entityName = getEntityName(entityType, data);

        expect(entityName).toBe(null);
    });

    it('handles nested data for an IMAGE entity', () => {
        const entityType = entityTypes.IMAGE;
        const data = {
            image: {
                id: 'sha256:d074a3df2ea6c6320a1d888f93995b92c1c072df2fd13814b42cce6109c5f685',
                name: {
                    fullName: 'us.gcr.io/ultra-current-825/struts-violations/visa-processor:latest',
                    __typename: 'ImageName'
                },
                __typename: 'Image'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual(
            'us.gcr.io/ultra-current-825/struts-violations/visa-processor:latest'
        );
    });

    it('handles nested data for a ROLE entity', () => {
        const entityType = entityTypes.ROLE;
        const data = {
            clusters: [
                {
                    id: '07a88749-1c93-4c64-98dc-a737dd8f2283',
                    k8srole: null,
                    __typename: 'Cluster'
                },
                {
                    id: 'de9e9702-f1f1-4a6e-b98a-1df994aeadc1',
                    k8srole: {
                        id: '017eefb7-6f72-11ea-a65a-42010a8a0161',
                        name: 'system:kube-dns',
                        __typename: 'K8SRole'
                    },
                    __typename: 'Cluster'
                }
            ]
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('system:kube-dns');
    });

    it('handles combined data for a CONTROL entity', () => {
        const entityType = entityTypes.CONTROL;
        const data = {
            control: {
                id: 'CIS_Docker_v1_2_0:1_2_1',
                name: '1.2.1',
                description: 'Ensure a separate partition for containers has been created',
                __typename: 'ComplianceControl'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual(
            '1.2.1 - Ensure a separate partition for containers has been created'
        );
    });

    it('handles nested data for a SUBJECT entity', () => {
        const entityType = entityTypes.SUBJECT;
        const data = {
            clusters: [
                {
                    id: '07a88749-1c93-4c64-98dc-a737dd8f2283',
                    subject: {
                        id: 'system:kube-controller-manager',
                        subject: {
                            name: 'system:kube-controller-manager',
                            __typename: 'Subject'
                        },
                        __typename: 'SubjectWithClusterID'
                    },
                    __typename: 'Cluster'
                },
                {
                    id: 'de9e9702-f1f1-4a6e-b98a-1df994aeadc1',
                    subject: {
                        id: 'system:kube-controller-manager',
                        subject: {
                            name: 'system:kube-controller-manager',
                            __typename: 'Subject'
                        },
                        __typename: 'SubjectWithClusterID'
                    },
                    __typename: 'Cluster'
                }
            ]
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('system:kube-controller-manager');
    });

    it('handles a SERVICE_ACCOUNT entity', () => {
        const entityType = entityTypes.SERVICE_ACCOUNT;
        const data = {
            serviceAccount: {
                id: '66853828-6f72-11ea-b000-42010a8a005a',
                name: 'admission-control',
                __typename: 'ServiceAccount'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('admission-control');
    });

    it('handles a SECRET entity', () => {
        const entityType = entityTypes.SECRET;
        const data = {
            secret: {
                id: '63dc443c-6f72-11ea-a65a-42010a8a0161',
                name: 'collector-stackrox',
                __typename: 'Secret'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('collector-stackrox');
    });

    it('handles a CLUSTER entity', () => {
        const entityType = entityTypes.CLUSTER;
        const data = {
            cluster: {
                id: '07a88749-1c93-4c64-98dc-a737dd8f2283',
                name: 'production',
                __typename: 'Cluster'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('production');
    });

    it('handles a COMPONENT entity, even when paired with an image', () => {
        const entityType = entityTypes.COMPONENT;
        const data = {
            image: {
                id: 'sha256:d074a3df2ea6c6320a1d888f93995b92c1c072df2fd13814b42cce6109c5f685',
                name: {
                    fullName: 'us.gcr.io/ultra-current-825/struts-violations/visa-processor:latest',
                    __typename: 'ImageName'
                },
                __typename: 'Image'
            },
            component: {
                id: 'bGlieG1sMg:Mi45LjErZGZzZzEtNStkZWI4dTQ',
                name: 'libxml2',
                __typename: 'EmbeddedImageScanComponent'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('libxml2');
    });

    it('handles a CVE entity', () => {
        const entityType = entityTypes.CVE;
        const data = {
            vulnerability: {
                id: 'CVE-2017-7376',
                name: 'CVE-2017-7376',
                cve: 'CVE-2017-7376',
                __typename: 'EmbeddedVulnerability'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('CVE-2017-7376');
    });

    it('handles a DEPLOYMENT entity', () => {
        const entityType = entityTypes.DEPLOYMENT;
        const data = {
            deployment: {
                id: '7ac0fabd-6f72-11ea-b000-42010a8a005a',
                name: 'visa-processor',
                __typename: 'Deployment'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('visa-processor');
    });

    it('handles nested data for a NAMESPACE entity', () => {
        const entityType = entityTypes.NAMESPACE;
        const data = {
            namespace: {
                metadata: {
                    name: 'stackrox',
                    id: '65acc658-6f72-11ea-b000-42010a8a005a',
                    __typename: 'NamespaceMetadata'
                },
                __typename: 'Namespace'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('stackrox');
    });

    it('handles a NODE entity', () => {
        const entityType = entityTypes.NODE;
        const data = {
            node: {
                id: '1bc856c3-6f72-11ea-b000-42010a8a005a',
                name: 'gke-setup-ops52f53-prod-default-pool-2fdbe4fa-08fh',
                __typename: 'Node'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('gke-setup-ops52f53-prod-default-pool-2fdbe4fa-08fh');
    });

    it('handles a POLICY entity', () => {
        const entityType = entityTypes.POLICY;
        const data = {
            policy: {
                id: '900990b5-60ef-44e5-b7f6-4a1f22215d7f',
                name: 'Apache Struts: CVE-2017-5638',
                __typename: 'Policy'
            }
        };

        const entityName = getEntityName(entityType, data);

        expect(entityName).toEqual('Apache Struts: CVE-2017-5638');
    });
});
