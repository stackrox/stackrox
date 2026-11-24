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
            queryString=""
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('displays unavailable scores and preserves numeric maxima', () => {
        const cases = {
            unavailable: {
                topCVSS: 'Not available',
                topNvdCVSS: null,
                expectedCvss: 'Not available',
                expectedNvd: 'Not available',
            },
            mixed: {
                topCVSS: 7.2,
                topNvdCVSS: 7.2,
                expectedCvss: '7.2',
                expectedNvd: '7.2',
            },
            missingNvd: {
                topCVSS: 6.8,
                topNvdCVSS: null,
                expectedCvss: '6.8',
                expectedNvd: 'Not available',
            },
            zero: {
                topCVSS: 0,
                topNvdCVSS: 0,
                expectedCvss: '0.0',
                expectedNvd: '0.0',
            },
        };
        const data = Object.entries(cases).map(([name, { topCVSS, topNvdCVSS }]) => ({
            cve: `CVE-${name}`,
            topCVSS,
            topNvdCVSS,
            topEpssProbability: 0,
            affectedImageCount: 1,
            firstDiscoveredInSystem: null,
            publishedOn: null,
            pendingExceptionCount: 0,
            knownExploit: false,
            knownRansomwareCampaignUse: false,
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
