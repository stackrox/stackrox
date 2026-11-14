import useMap from 'hooks/useMap';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';

import CVEsTable, { defaultSortOption } from './CVEsTable';

function AggregateScoreTable() {
    const selectedCves = useMap();

    return (
        <CVEsTable
            querySearchFilter={{}}
            isFiltered={false}
            pagination={{ page: 1, perPage: 20 }}
            selectedCves={selectedCves}
            createRowActions={() => []}
            sortOption={defaultSortOption}
            getSortParams={() => undefined}
            onClearFilters={() => {}}
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('handles unavailable top CVSS text in API responses and preserves numeric maxima', () => {
        const cases = {
            unavailable: { topCVSS: 'Not available', scores: [null], expected: 'Not available' },
            otherText: { topCVSS: 'No score', scores: [null], expected: 'Not available' },
            mixed: { topCVSS: 7.2, scores: [7.2, 5.8, null], expected: '7.2 (V3)' },
            zero: { topCVSS: 0, scores: [0], expected: '0.0 (V3)' },
        };
        const nodeCVEs = Object.entries(cases).map(([name, { topCVSS, scores }]) => ({
            cve: `CVE-${name}`,
            affectedNodeCountBySeverity: {
                critical: { total: 0 },
                important: { total: 0 },
                moderate: { total: 0 },
                low: { total: 0 },
                unknown: { total: 1 },
            },
            topCVSS,
            affectedNodeCount: 1,
            firstDiscoveredInSystem: null,
            distroTuples: scores.map((cvss) => ({
                summary: 'Example vulnerability',
                operatingSystem: 'rhel:9',
                cvss,
                scoreVersion: 'V3',
            })),
        }));
        cy.intercept('POST', graphqlUrl('getNodeCVEs'), {
            body: { data: { nodeCVEs } },
        });
        cy.intercept('POST', graphqlUrl('getTotalNodeCount'), {
            body: { data: { nodeCount: 3 } },
        });

        cy.mount(
            <ComponentTestProvider>
                <AggregateScoreTable />
            </ComponentTestProvider>
        );

        Object.entries(cases).forEach(([name, { expected }]) => {
            cy.findByRole('row', { name: new RegExp(`CVE-${name}`) }).within(() => {
                cy.get('td[data-label="Top CVSS"]').should('have.text', expected);
            });
        });
    });
});
