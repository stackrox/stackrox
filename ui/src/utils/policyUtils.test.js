// system under test (SUT)
import { getPolicySeverityCounts, sortDeploymentsByPolicyViolations } from './policyUtils';

describe('policyUtils', () => {
    describe('getPolicySeverityCounts', () => {
        it('should return all 0 counts when no policies passed in', () => {
            const failingPolicies = [];

            const severity = getPolicySeverityCounts(failingPolicies);
            expect(severity.critical).toEqual(0);
            expect(severity.high).toEqual(0);
            expect(severity.medium).toEqual(0);
            expect(severity.low).toEqual(0);
        });

        it('should count each type of policy', () => {
            const failingPolicies = [
                {
                    id: 'c09f8da1-6111-4ca0-8f49-294a76c65112',
                    severity: 'MEDIUM_SEVERITY',
                },
                {
                    id: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
                    severity: 'HIGH_SEVERITY',
                },
                {
                    id: 'a09f8da1-6111-4ca0-8f49-294a76c65110',
                    severity: 'LOW_SEVERITY',
                },
                {
                    id: 'e09f8da1-6111-4ca0-8f49-294a76c65119',
                    severity: 'CRITICAL_SEVERITY',
                },
            ];

            const severity = getPolicySeverityCounts(failingPolicies);

            expect(severity.critical).toEqual(1);
            expect(severity.high).toEqual(1);
            expect(severity.medium).toEqual(1);
            expect(severity.low).toEqual(1);
        });

        it('should ignore policies with an unrecognized severity', () => {
            const failingPolicies = [
                {
                    id: 'c09f8da1-6111-4ca0-8f49-294a76c65112',
                    severity: 'MEDIUM_SEVERITY',
                },
                {
                    id: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
                    severity: 'UNKNOWN_SEVERITY',
                },
                {
                    id: 'a09f8da1-6111-4ca0-8f49-294a76c65110',
                    severity: 'LOW_SEVERITY',
                },
                {
                    id: 'e09f8da1-6111-4ca0-8f49-294a76c65119',
                    severity: 'CRITICAL_SEVERITY',
                },
            ];

            const severity = getPolicySeverityCounts(failingPolicies);

            expect(severity.critical).toEqual(1);
            expect(severity.high).toEqual(0);
            expect(severity.medium).toEqual(1);
            expect(severity.low).toEqual(1);
        });

        it('should ignore policies that do not have a severity property', () => {
            const failingPolicies = [
                {
                    id: 'c09f8da1-6111-4ca0-8f49-294a76c65112',
                    severity: 'MEDIUM_SEVERITY',
                },
                {
                    id: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
                },
                {
                    id: 'a09f8da1-6111-4ca0-8f49-294a76c65110',
                },
                {
                    id: 'e09f8da1-6111-4ca0-8f49-294a76c65119',
                    severity: 'CRITICAL_SEVERITY',
                },
            ];

            const severity = getPolicySeverityCounts(failingPolicies);

            expect(severity.critical).toEqual(1);
            expect(severity.high).toEqual(0);
            expect(severity.medium).toEqual(1);
            expect(severity.low).toEqual(0);
        });

        it('should count mulitple instances for each type of policy', () => {
            const failingPolicies = [
                {
                    id: 'a3eb6dbe-e9ca-451a-919b-216cf7ee11f5',
                    severity: 'LOW_SEVERITY',
                },
                {
                    id: 'c09f8da1-6111-4ca0-8f49-294a76c65112',
                    severity: 'MEDIUM_SEVERITY',
                },
                {
                    id: 'f09f8da1-6111-4ca0-8f49-294a76c65115',
                    severity: 'HIGH_SEVERITY',
                },
                {
                    id: '80267b36-2182-4fb3-8b53-e80c031f4ad8',
                    severity: 'CRITICAL_SEVERITY',
                },
                {
                    id: 'a09f8da1-6111-4ca0-8f49-294a76c65110',
                    severity: 'LOW_SEVERITY',
                },
                {
                    id: '900990b5-60ef-44e5-b7f6-4a1f22215d7f',
                    severity: 'HIGH_SEVERITY',
                },
                {
                    id: 'e09f8da1-6111-4ca0-8f49-294a76c65119',
                    severity: 'CRITICAL_SEVERITY',
                },
                {
                    id: '3a98be1e-d427-41ba-ad60-994e848a5554',
                    severity: 'MEDIUM_SEVERITY',
                },
            ];

            const severity = getPolicySeverityCounts(failingPolicies);

            expect(severity.critical).toEqual(2);
            expect(severity.high).toEqual(2);
            expect(severity.medium).toEqual(2);
            expect(severity.low).toEqual(2);
        });
    });

    describe('sortPoliciesBySevereViolations', () => {
        it('should return an empty array when passed an empty array', () => {
            const deployments = [];

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments).toEqual([]);
        });

        it('should return deployment sorted by failing policy severity first', () => {
            const deployments = getMinimalExample();

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments[0]).toEqual(deployments[2]);
            expect(sortedDeployments[1]).toEqual(deployments[3]);
            expect(sortedDeployments[2]).toEqual(deployments[0]);
            expect(sortedDeployments[3]).toEqual(deployments[1]);
        });

        it('should sorts higher severity violations above larger numbers of lower severity violations', () => {
            const deployments = getFirstLopsidedExample();

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments[0]).toEqual(deployments[1]);
            expect(sortedDeployments[1]).toEqual(deployments[2]);
            expect(sortedDeployments[2]).toEqual(deployments[3]);
            expect(sortedDeployments[3]).toEqual(deployments[0]);
        });

        it('should sorts higher severity violations above arbitrarily large numbers of lower severity violations', () => {
            const deployments = getSecondLopsidedExample();

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments[0]).toEqual(deployments[1]);
            expect(sortedDeployments[1]).toEqual(deployments[0]);
            expect(sortedDeployments[2]).toEqual(deployments[3]);
            expect(sortedDeployments[3]).toEqual(deployments[2]);
        });

        it('should sorts ties at one severity by the next lower level', () => {
            const deployments = getFirstTiebreakerExample();

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments[0]).toEqual(deployments[1]);
            expect(sortedDeployments[1]).toEqual(deployments[3]);
            expect(sortedDeployments[2]).toEqual(deployments[2]);
            expect(sortedDeployments[3]).toEqual(deployments[0]);
        });

        it('should sorts mutiple ties at one severity by the closest level applicable', () => {
            const deployments = getSecondTiebreakerExample();

            const sortedDeployments = sortDeploymentsByPolicyViolations(deployments);

            expect(sortedDeployments[0]).toEqual(deployments[3]);
            expect(sortedDeployments[1]).toEqual(deployments[2]);
            expect(sortedDeployments[2]).toEqual(deployments[1]);
            expect(sortedDeployments[3]).toEqual(deployments[0]);
        });
    });
});

function getMinimalExample() {
    return [
        {
            id: '8bb59a49-0ae9-11ea-9e69-025000000001',
            name: 'compose',
            policySeverityCounts: { critical: 0, high: 0, medium: 1, low: 0 },
        },
        {
            id: '6687eb8a-0ae9-11ea-9e69-025000000001',
            name: 'coredns',
            policySeverityCounts: { critical: 0, high: 0, medium: 0, low: 1 },
        },
        {
            id: '66bb3420-0ae9-11ea-9e69-025000000001',
            name: 'kube-proxy',
            policySeverityCounts: { critical: 1, high: 0, medium: 0, low: 0 },
        },
        {
            id: '8bb0dd1e-0ae9-11ea-9e69-025000000001',
            name: 'compose-api',
            policySeverityCounts: { critical: 0, high: 1, medium: 0, low: 0 },
        },
    ];
}

function getFirstLopsidedExample() {
    return [
        {
            id: '8bb59a49-0ae9-11ea-9e69-025000000001',
            name: 'compose',
            policySeverityCounts: { critical: 0, high: 0, medium: 1, low: 4000 },
        },
        {
            id: '6687eb8a-0ae9-11ea-9e69-025000000001',
            name: 'coredns',
            policySeverityCounts: { critical: 1, high: 0, medium: 0, low: 1 },
        },
        {
            id: '66bb3420-0ae9-11ea-9e69-025000000001',
            name: 'kube-proxy',
            policySeverityCounts: { critical: 0, high: 100, medium: 0, low: 0 },
        },
        {
            id: '8bb0dd1e-0ae9-11ea-9e69-025000000001',
            name: 'compose-api',
            policySeverityCounts: { critical: 0, high: 1, medium: 1, low: 0 },
        },
    ];
}

function getSecondLopsidedExample() {
    return [
        {
            id: '8bb59a49-0ae9-11ea-9e69-025000000001',
            name: 'compose',
            policySeverityCounts: { critical: 0, high: 99, medium: 214, low: 2000 },
        },
        {
            id: '6687eb8a-0ae9-11ea-9e69-025000000001',
            name: 'coredns',
            policySeverityCounts: { critical: 1, high: 0, medium: 114, low: 1000 },
        },
        {
            id: '66bb3420-0ae9-11ea-9e69-025000000001',
            name: 'kube-proxy',
            policySeverityCounts: { critical: 0, high: 0, medium: 313, low: 4000 },
        },
        {
            id: '8bb0dd1e-0ae9-11ea-9e69-025000000001',
            name: 'compose-api',
            policySeverityCounts: { critical: 0, high: 0, medium: 314, low: 3000 },
        },
    ];
}

function getFirstTiebreakerExample() {
    return [
        {
            id: '8bb59a49-0ae9-11ea-9e69-025000000001',
            name: 'compose',
            policySeverityCounts: { critical: 0, high: 3, medium: 99, low: 0 },
        },
        {
            id: '6687eb8a-0ae9-11ea-9e69-025000000001',
            name: 'coredns',
            policySeverityCounts: { critical: 1, high: 2, medium: 0, low: 0 },
        },
        {
            id: '66bb3420-0ae9-11ea-9e69-025000000001',
            name: 'kube-proxy',
            policySeverityCounts: { critical: 0, high: 3, medium: 100, low: 0 },
        },
        {
            id: '8bb0dd1e-0ae9-11ea-9e69-025000000001',
            name: 'compose-api',
            policySeverityCounts: { critical: 1, high: 1, medium: 0, low: 0 },
        },
    ];
}

function getSecondTiebreakerExample() {
    return [
        {
            id: '8bb59a49-0ae9-11ea-9e69-025000000001',
            name: 'compose',
            policySeverityCounts: { critical: 0, high: 3, medium: 0, low: 41 },
        },
        {
            id: '6687eb8a-0ae9-11ea-9e69-025000000001',
            name: 'coredns',
            policySeverityCounts: { critical: 0, high: 3, medium: 0, low: 42 },
        },
        {
            id: '66bb3420-0ae9-11ea-9e69-025000000001',
            name: 'kube-proxy',
            policySeverityCounts: { critical: 0, high: 3, medium: 2, low: 0 },
        },
        {
            id: '8bb0dd1e-0ae9-11ea-9e69-025000000001',
            name: 'compose-api',
            policySeverityCounts: { critical: 0, high: 3, medium: 3, low: 0 },
        },
    ];
}
