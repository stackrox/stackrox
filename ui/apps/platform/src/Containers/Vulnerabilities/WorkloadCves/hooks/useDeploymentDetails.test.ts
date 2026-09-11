import type { Deployment } from 'types/deployment.proto';

import { mapDeploymentToDetails } from './useDeploymentDetails';

const deployment = {
    id: 'dep-1',
    name: 'central',
    type: 'Deployment',
    namespace: 'stackrox',
    replicas: '3',
    labels: { app: 'central', owner: 'platform' },
    created: '2024-01-15T10:30:00Z',
    clusterId: 'cluster-1',
    clusterName: 'remote',
    annotations: { note: 'qa' },
    serviceAccount: 'central',
} as unknown as Deployment;

describe('mapDeploymentToDetails', () => {
    it('maps REST deployment fields onto the WorkloadCves details shape', () => {
        expect(mapDeploymentToDetails(deployment)).toEqual({
            id: 'dep-1',
            name: 'central',
            cluster: { id: 'cluster-1', name: 'remote' },
            namespace: 'stackrox',
            replicas: 3,
            created: '2024-01-15T10:30:00Z',
            serviceAccount: 'central',
            type: 'Deployment',
            labels: [
                { key: 'app', value: 'central' },
                { key: 'owner', value: 'platform' },
            ],
            annotations: [{ key: 'note', value: 'qa' }],
        });
    });
});
