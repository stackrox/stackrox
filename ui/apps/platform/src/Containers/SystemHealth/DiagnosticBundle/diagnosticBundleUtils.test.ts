import { getQueryString } from './diagnosticBundleUtils';

describe('Diagnostic Bundle dialog box', () => {
    describe('query string', () => {
        it('should not have params for initial defaults', () => {
            const expected = '';
            const received = getQueryString({
                selectedClusterNames: [],
                startingTimeIso: null,
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        it('should have param for one selected cluster', () => {
            const expected = '?cluster=abbot';
            const received = getQueryString({
                selectedClusterNames: ['abbot'],
                startingTimeIso: null,
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        // qs encodeValuesOnly option replaces colon with %3A in starting time.

        it('should have a param for valid starting time', () => {
            const expected = '?since=2020-10-20T20%3A21%3A00.000Z';
            const received = getQueryString({
                selectedClusterNames: [],
                startingTimeIso: '2020-10-20T20:21:00.000Z',
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        it('should have params for one selected cluster and valid starting time', () => {
            const expected = '?since=2020-10-20T20%3A21%3A22.000Z&cluster=costello';
            const received = getQueryString({
                selectedClusterNames: ['costello'],
                startingTimeIso: '2020-10-20T20:21:22.000Z',
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        it('should have params for two selected clusters and valid starting time', () => {
            const expected = '?since=2020-10-20T20%3A21%3A22.345Z&cluster=costello&cluster=abbot';
            const received = getQueryString({
                selectedClusterNames: ['costello', 'abbot'],
                startingTimeIso: '2020-10-20T20:21:22.345Z',
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        it('should have param for database-only diagnostics', () => {
            const expected = '?database-only=true';
            const received = getQueryString({
                selectedClusterNames: [],
                startingTimeIso: null,
                isDatabaseDiagnosticsOnly: true,
                includeComplianceOperatorResources: false,
            });

            expect(received).toBe(expected);
        });

        it('should have param for compliance operator resources', () => {
            const expected = '?compliance-operator=true';
            const received = getQueryString({
                selectedClusterNames: [],
                startingTimeIso: null,
                isDatabaseDiagnosticsOnly: false,
                includeComplianceOperatorResources: true,
            });

            expect(received).toBe(expected);
        });

        // UI disables fields when database-only is selected, but URL can still include them. backend ignores irrelevant ones.
        it('should have all params when all options are selected', () => {
            const expected =
                '?database-only=true&compliance-operator=true&since=2020-10-20T20%3A21%3A22.000Z&cluster=test-cluster';
            const received = getQueryString({
                selectedClusterNames: ['test-cluster'],
                startingTimeIso: '2020-10-20T20:21:22.000Z',
                isDatabaseDiagnosticsOnly: true,
                includeComplianceOperatorResources: true,
            });

            expect(received).toBe(expected);
        });
    });
});
