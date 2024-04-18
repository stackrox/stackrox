import {
    getNodeEntityPagePath,
    getPlatformEntityPagePath,
    getWorkloadEntityPagePath,
} from './searchUtils';

const workloadUrlBase = '/main/vulnerabilities/workload-cves';

describe('getWorkloadEntityPagePath', () => {
    it('should return the correct path for CVE entity', () => {
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'OBSERVED')).toEqual(
            `${workloadUrlBase}/cves/CVE-123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'DEFERRED')).toEqual(
            `${workloadUrlBase}/cves/CVE-123-456?vulnerabilityState=DEFERRED`
        );
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'FALSE_POSITIVE')).toEqual(
            `${workloadUrlBase}/cves/CVE-123-456?vulnerabilityState=FALSE_POSITIVE`
        );

        expect(
            getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'OBSERVED', {
                s: { SEVERITY: ['CRITICAL', 'IMPORTANT'], FIXABLE: [], NAMESPACE: ['stackrox'] },
            })
        ).toEqual(
            `${workloadUrlBase}/cves/CVE-123-456?s[SEVERITY][0]=CRITICAL&s[SEVERITY][1]=IMPORTANT&s[NAMESPACE][0]=stackrox&vulnerabilityState=OBSERVED`
        );
    });

    it('should return the correct path for Image entity', () => {
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'OBSERVED')).toEqual(
            `${workloadUrlBase}/images/sha256:123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'DEFERRED')).toEqual(
            `${workloadUrlBase}/images/sha256:123-456?vulnerabilityState=DEFERRED`
        );
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'FALSE_POSITIVE')).toEqual(
            `${workloadUrlBase}/images/sha256:123-456?vulnerabilityState=FALSE_POSITIVE`
        );

        expect(
            getWorkloadEntityPagePath('Image', 'sha256:123-456', 'OBSERVED', {
                s: { SEVERITY: ['CRITICAL', 'IMPORTANT'], FIXABLE: [], NAMESPACE: ['stackrox'] },
            })
        ).toEqual(
            `${workloadUrlBase}/images/sha256:123-456?s[SEVERITY][0]=CRITICAL&s[SEVERITY][1]=IMPORTANT&s[NAMESPACE][0]=stackrox&vulnerabilityState=OBSERVED`
        );
    });

    it('should return the correct path for Deployment entity', () => {
        expect(getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'OBSERVED')).toEqual(
            `${workloadUrlBase}/deployments/deployment-123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'DEFERRED')).toEqual(
            `${workloadUrlBase}/deployments/deployment-123-456?vulnerabilityState=DEFERRED`
        );
        expect(
            getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'FALSE_POSITIVE')
        ).toEqual(
            `${workloadUrlBase}/deployments/deployment-123-456?vulnerabilityState=FALSE_POSITIVE`
        );

        expect(
            getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'OBSERVED', {
                s: { SEVERITY: ['CRITICAL', 'IMPORTANT'], FIXABLE: [], NAMESPACE: ['stackrox'] },
            })
        ).toEqual(
            `${workloadUrlBase}/deployments/deployment-123-456?s[SEVERITY][0]=CRITICAL&s[SEVERITY][1]=IMPORTANT&s[NAMESPACE][0]=stackrox&vulnerabilityState=OBSERVED`
        );
    });
});

const platformUrlBase = '/main/vulnerabilities/platform-cves';

describe('getPlatformEntityPagePath', () => {
    it('should return the correct path for CVE entity', () => {
        expect(getPlatformEntityPagePath('CVE', 'CVE-123-456')).toEqual(
            `${platformUrlBase}/cves/CVE-123-456`
        );

        expect(
            getPlatformEntityPagePath('CVE', 'CVE-123-456', { s: { SEVERITY: ['CRITICAL'] } })
        ).toEqual(`${platformUrlBase}/cves/CVE-123-456?s[SEVERITY][0]=CRITICAL`);
    });

    it('should return the correct path for Cluster entity', () => {
        expect(getPlatformEntityPagePath('Cluster', 'cluster-123-456')).toEqual(
            `${platformUrlBase}/clusters/cluster-123-456`
        );

        expect(
            getPlatformEntityPagePath('Cluster', 'cluster-123-456', {
                s: { SEVERITY: ['CRITICAL'], NAMESPACE: ['stackrox'] },
            })
        ).toEqual(
            `${platformUrlBase}/clusters/cluster-123-456?s[SEVERITY][0]=CRITICAL&s[NAMESPACE][0]=stackrox`
        );
    });
});

const nodeUrlBase = '/main/vulnerabilities/node-cves';

describe('getNodeEntityPagePath', () => {
    it('should return the correct path for CVE entity', () => {
        expect(getNodeEntityPagePath('CVE', 'CVE-123-456')).toEqual(
            `${nodeUrlBase}/cves/CVE-123-456`
        );

        expect(
            getNodeEntityPagePath('CVE', 'CVE-123-456', { s: { SEVERITY: ['CRITICAL'] } })
        ).toEqual(`${nodeUrlBase}/cves/CVE-123-456?s[SEVERITY][0]=CRITICAL`);
    });

    it('should return the correct path for Node entity', () => {
        expect(getNodeEntityPagePath('Node', 'node-123-456')).toEqual(
            `${nodeUrlBase}/nodes/node-123-456`
        );

        expect(
            getNodeEntityPagePath('Node', 'node-123-456', {
                s: { SEVERITY: ['CRITICAL'], NAMESPACE: ['stackrox'] },
            })
        ).toEqual(
            `${nodeUrlBase}/nodes/node-123-456?s[SEVERITY][0]=CRITICAL&s[NAMESPACE][0]=stackrox`
        );
    });
});
