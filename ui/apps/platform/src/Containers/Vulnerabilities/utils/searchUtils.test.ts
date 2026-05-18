import {
    getAppliedSeverities,
    getHiddenSeverities,
    getNodeEntityPagePath,
    getPlatformEntityPagePath,
    getWorkloadEntityPagePath,
    normalizeSearchFilterKeys,
    parseQuerySearchFilter,
} from './searchUtils';

describe('normalizeSearchFilterKeys', () => {
    it('should rename SEVERITY to Severity', () => {
        const filter = { SEVERITY: ['Critical'], Namespace: ['stackrox'] };
        const result = normalizeSearchFilterKeys(filter);
        expect(result).toEqual({ Severity: ['Critical'], Namespace: ['stackrox'] });
    });

    it('should not modify a filter that already uses canonical keys', () => {
        const filter = { Severity: ['Critical'], Fixable: ['Fixable'] };
        const result = normalizeSearchFilterKeys(filter);
        expect(result).toBe(filter);
    });

    it('should prefer an existing canonical key', () => {
        const filter = { SEVERITY: ['Low'], Severity: ['Critical'] };
        const result = normalizeSearchFilterKeys(filter);
        expect(result).toEqual({ Severity: ['Critical'] });
    });

    it('should handle an empty filter', () => {
        const filter = {};
        const result = normalizeSearchFilterKeys(filter);
        expect(result).toBe(filter);
    });

    it('should leave unrelated keys untouched', () => {
        const filter = { SEVERITY: ['Critical'], Namespace: ['stackrox'], CVE: ['CVE-2024-1234'] };
        const result = normalizeSearchFilterKeys(filter);
        expect(result).toEqual({
            Severity: ['Critical'],
            Namespace: ['stackrox'],
            CVE: ['CVE-2024-1234'],
        });
    });
});

describe('parseQuerySearchFilter', () => {
    it('should convert Severity labels to backend severity enums', () => {
        const result = parseQuerySearchFilter({ Severity: ['Critical', 'Important'] });
        expect(result.Severity).toEqual([
            'CRITICAL_VULNERABILITY_SEVERITY',
            'IMPORTANT_VULNERABILITY_SEVERITY',
        ]);
    });

    it('should handle legacy SEVERITY key via case-insensitive lookup', () => {
        const result = parseQuerySearchFilter({ SEVERITY: ['Critical', 'Low'] });
        expect(result.Severity).toEqual([
            'CRITICAL_VULNERABILITY_SEVERITY',
            'LOW_VULNERABILITY_SEVERITY',
        ]);
    });

    it('should filter out invalid severity labels', () => {
        const result = parseQuerySearchFilter({ Severity: ['Critical', 'InvalidValue'] });
        expect(result.Severity).toEqual(['CRITICAL_VULNERABILITY_SEVERITY']);
    });

    it('should pass through unrelated keys unchanged', () => {
        const result = parseQuerySearchFilter({ Namespace: ['stackrox'] });
        expect(result.Namespace).toEqual(['stackrox']);
        expect(result.Severity).toBeUndefined();
    });

    it('should handle empty search filter', () => {
        const result = parseQuerySearchFilter({});
        expect(result.Severity).toBeUndefined();
    });
});

describe('getAppliedSeverities', () => {
    it('should return severity labels from Severity key', () => {
        expect(getAppliedSeverities({ Severity: ['Critical', 'Moderate'] })).toEqual([
            'Critical',
            'Moderate',
        ]);
    });

    it('should return severity labels from legacy SEVERITY key', () => {
        expect(getAppliedSeverities({ SEVERITY: ['Critical', 'Low'] })).toEqual([
            'Critical',
            'Low',
        ]);
    });

    it('should filter out non-severity values', () => {
        expect(getAppliedSeverities({ Severity: ['Critical', 'NotASeverity'] })).toEqual([
            'Critical',
        ]);
    });

    it('should return empty array when no severity is present', () => {
        expect(getAppliedSeverities({})).toEqual([]);
    });
});

describe('getHiddenSeverities', () => {
    it('should return empty set when no severity filter is applied', () => {
        expect(getHiddenSeverities({})).toEqual(new Set([]));
    });

    it('should return hidden severities based on applied filter', () => {
        const result = getHiddenSeverities({
            Severity: ['CRITICAL_VULNERABILITY_SEVERITY'],
        });
        expect(result).toEqual(
            new Set([
                'IMPORTANT_VULNERABILITY_SEVERITY',
                'MODERATE_VULNERABILITY_SEVERITY',
                'LOW_VULNERABILITY_SEVERITY',
                'UNKNOWN_VULNERABILITY_SEVERITY',
            ])
        );
    });
});

describe('getWorkloadEntityPagePath', () => {
    it('should return the correct path for CVE entity', () => {
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'OBSERVED')).toEqual(
            `cves/CVE-123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'DEFERRED')).toEqual(
            `cves/CVE-123-456?vulnerabilityState=DEFERRED`
        );
        expect(getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'FALSE_POSITIVE')).toEqual(
            `cves/CVE-123-456?vulnerabilityState=FALSE_POSITIVE`
        );

        expect(
            getWorkloadEntityPagePath('CVE', 'CVE-123-456', 'OBSERVED', {
                s: { Severity: ['Critical', 'Important'], Fixable: [], Namespace: ['stackrox'] },
            })
        ).toEqual(
            `cves/CVE-123-456?s[Severity][0]=Critical&s[Severity][1]=Important&s[Namespace][0]=stackrox&vulnerabilityState=OBSERVED`
        );
    });

    it('should return the correct path for Image entity', () => {
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'OBSERVED')).toEqual(
            `images/sha256:123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'DEFERRED')).toEqual(
            `images/sha256:123-456?vulnerabilityState=DEFERRED`
        );
        expect(getWorkloadEntityPagePath('Image', 'sha256:123-456', 'FALSE_POSITIVE')).toEqual(
            `images/sha256:123-456?vulnerabilityState=FALSE_POSITIVE`
        );

        expect(
            getWorkloadEntityPagePath('Image', 'sha256:123-456', 'OBSERVED', {
                s: { Severity: ['Critical', 'Important'], Fixable: [], Namespace: ['stackrox'] },
            })
        ).toEqual(
            `images/sha256:123-456?s[Severity][0]=Critical&s[Severity][1]=Important&s[Namespace][0]=stackrox&vulnerabilityState=OBSERVED`
        );
    });

    it('should return the correct path for Deployment entity', () => {
        expect(getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'OBSERVED')).toEqual(
            `deployments/deployment-123-456?vulnerabilityState=OBSERVED`
        );
        expect(getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'DEFERRED')).toEqual(
            `deployments/deployment-123-456?vulnerabilityState=DEFERRED`
        );
        expect(
            getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'FALSE_POSITIVE')
        ).toEqual(`deployments/deployment-123-456?vulnerabilityState=FALSE_POSITIVE`);

        expect(
            getWorkloadEntityPagePath('Deployment', 'deployment-123-456', 'OBSERVED', {
                s: { Severity: ['Critical', 'Important'], Fixable: [], Namespace: ['stackrox'] },
            })
        ).toEqual(
            `deployments/deployment-123-456?s[Severity][0]=Critical&s[Severity][1]=Important&s[Namespace][0]=stackrox&vulnerabilityState=OBSERVED`
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
            getPlatformEntityPagePath('CVE', 'CVE-123-456', { s: { Severity: ['Critical'] } })
        ).toEqual(`${platformUrlBase}/cves/CVE-123-456?s[Severity][0]=Critical`);
    });

    it('should return the correct path for Cluster entity', () => {
        expect(getPlatformEntityPagePath('Cluster', 'cluster-123-456')).toEqual(
            `${platformUrlBase}/clusters/cluster-123-456`
        );

        expect(
            getPlatformEntityPagePath('Cluster', 'cluster-123-456', {
                s: { Severity: ['Critical'], Namespace: ['stackrox'] },
            })
        ).toEqual(
            `${platformUrlBase}/clusters/cluster-123-456?s[Severity][0]=Critical&s[Namespace][0]=stackrox`
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
            getNodeEntityPagePath('CVE', 'CVE-123-456', { s: { Severity: ['Critical'] } })
        ).toEqual(`${nodeUrlBase}/cves/CVE-123-456?s[Severity][0]=Critical`);
    });

    it('should return the correct path for Node entity', () => {
        expect(getNodeEntityPagePath('Node', 'node-123-456')).toEqual(
            `${nodeUrlBase}/nodes/node-123-456`
        );

        expect(
            getNodeEntityPagePath('Node', 'node-123-456', {
                s: { Severity: ['Critical'], Namespace: ['stackrox'] },
            })
        ).toEqual(
            `${nodeUrlBase}/nodes/node-123-456?s[Severity][0]=Critical&s[Namespace][0]=stackrox`
        );
    });
});
