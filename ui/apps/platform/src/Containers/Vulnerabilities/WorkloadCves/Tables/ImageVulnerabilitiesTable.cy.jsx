import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import { MetadataContext } from 'hooks/useMetadata';
import useMap from 'hooks/useMap';
import { useManagedColumns } from 'hooks/useManagedColumns';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import { WorkloadCveViewContext } from '../WorkloadCveViewContext';
import ImageVulnerabilitiesTable, { defaultColumns } from './ImageVulnerabilitiesTable';

function InstanceScoreTable({ data }) {
    const selectedCves = useMap();
    const { columns } = useManagedColumns('InstanceScoreTable', defaultColumns);

    return (
        <ImageVulnerabilitiesTable
            tableState={{ type: 'COMPLETE', data }}
            getSortParams={() => undefined}
            isFiltered={false}
            selectedCves={selectedCves}
            vulnerabilityState="OBSERVED"
            onClearFilters={() => {}}
            tableConfig={columns}
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('preserves missing NVD scores even when a score version is present', () => {
        const scores = { unavailable: null, zero: 0, scored: 7.234 };
        const data = Object.entries(scores).map(([name, nvdCvss]) => ({
            cve: `CVE-${name}`,
            severity: 'IMPORTANT_VULNERABILITY_SEVERITY',
            summary: '',
            cvss: 7.2,
            scoreVersion: 'V3',
            nvdCvss,
            nvdScoreVersion: 'V3',
            cveBaseInfo: {},
            discoveredAtImage: null,
            publishedOn: null,
            pendingExceptionCount: 0,
            imageComponents: [],
        }));

        cy.mount(
            <ComponentTestProvider>
                <FeatureFlagsContext.Provider value={{ isFeatureFlagEnabled: () => false }}>
                    <MetadataContext.Provider value={{ metadata: { version: '4.10.0' } }}>
                        <WorkloadCveViewContext.Provider
                            value={{ urlBuilder: { cveDetails: (cve) => `/cves/${cve}` } }}
                        >
                            <InstanceScoreTable data={data} />
                        </WorkloadCveViewContext.Provider>
                    </MetadataContext.Provider>
                </FeatureFlagsContext.Provider>
            </ComponentTestProvider>
        );

        cy.findByRole('row', { name: /CVE-unavailable/ }).within(() => {
            cy.get('td[data-label="NVD CVSS"]').should('have.text', 'Not available');
        });
        cy.findByRole('row', { name: /CVE-zero/ }).within(() => {
            cy.get('td[data-label="NVD CVSS"]').should('have.text', '0.0 (V3)');
        });
        cy.findByRole('row', { name: /CVE-scored/ }).within(() => {
            cy.get('td[data-label="NVD CVSS"]').should('have.text', '7.2 (V3)');
        });
    });
});
