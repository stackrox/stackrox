import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import { useManagedColumns } from 'hooks/useManagedColumns';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import { WorkloadCveViewContext } from '../WorkloadCveViewContext';
import WorkloadCVEOverviewTable, { defaultColumns } from './WorkloadCVEOverviewTable';

function AggregateScoreTable({ data }) {
    const selectedCves = useMap();
    const { columns } = useManagedColumns('AggregateScoreTable', defaultColumns);

    return (
        <WorkloadCVEOverviewTable
            tableState={{ type: 'COMPLETE', data }}
            unfilteredImageCount={3}
            getSortParams={() => undefined}
            isFiltered={false}
            selectedCves={selectedCves}
            vulnerabilityState="OBSERVED"
            onClearFilters={() => {}}
            columnVisibilityState={columns}
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('displays unavailable aggregate text and preserves numeric maxima', () => {
        const cases = {
            unavailable: {
                topCVSS: 'Not available',
                topNvdCVSS: 'No score',
                scores: [null],
                expectedCvss: 'Not available',
                expectedNvd: 'Not available',
            },
            mixed: {
                topCVSS: 7.2,
                topNvdCVSS: 7.2,
                scores: [7.2, 5.8, null],
                expectedCvss: '7.2 (V3)',
                expectedNvd: '7.2 (V3)',
            },
            missingNvd: {
                topCVSS: 7.2,
                topNvdCVSS: 'Not available',
                scores: [7.2, null],
                expectedCvss: '7.2 (V3)',
                expectedNvd: 'Not available',
            },
            zero: {
                topCVSS: 0,
                topNvdCVSS: 0,
                scores: [0],
                expectedCvss: '0.0 (V3)',
                expectedNvd: '0.0 (V3)',
            },
        };
        const data = Object.entries(cases).map(([name, { topCVSS, topNvdCVSS, scores }]) => ({
            cve: `CVE-${name}`,
            affectedImageCountBySeverity: {
                critical: { total: 0 },
                important: { total: 0 },
                moderate: { total: 0 },
                low: { total: 0 },
                unknown: { total: 1 },
            },
            topCVSS,
            topNvdCVSS,
            affectedImageCount: 1,
            firstDiscoveredInSystem: null,
            publishedOn: null,
            pendingExceptionCount: 0,
            distroTuples: scores.map((cvss) => ({
                summary: 'Example vulnerability',
                operatingSystem: 'rhel:9',
                cvss,
                scoreVersion: 'V3',
                nvdCvss: typeof topNvdCVSS === 'number' ? cvss : null,
                nvdScoreVersion: 'V3',
                cveBaseInfo: {},
            })),
        }));

        cy.mount(
            <ComponentTestProvider>
                <FeatureFlagsContext.Provider value={{ isFeatureFlagEnabled: () => false }}>
                    <WorkloadCveViewContext.Provider
                        value={{ urlBuilder: { cveDetails: (cve) => `/cves/${cve}` } }}
                    >
                        <AggregateScoreTable data={data} />
                    </WorkloadCveViewContext.Provider>
                </FeatureFlagsContext.Provider>
            </ComponentTestProvider>
        );

        Object.entries(cases).forEach(([name, { expectedCvss, expectedNvd }]) => {
            cy.findByRole('row', { name: new RegExp(`CVE-${name}`) }).within(() => {
                cy.get('td[data-label="Top CVSS"]').should('have.text', expectedCvss);
                cy.get('td[data-label="Top NVD CVSS"]').should('have.text', expectedNvd);
            });
        });
    });
});
