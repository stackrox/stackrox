import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { graphqlUrl } from 'test-utils/apiEndpoints';
import workflowStateContext from 'Containers/workflowStateContext';
import useCases from 'constants/useCaseTypes';
import { WorkflowState } from 'utils/WorkflowState';

import MostCommonVulnerabilities from './MostCommonVulnerabilities';

describe(Cypress.spec.relative, () => {
    it('displays missing and available instance scores together without crashing', () => {
        const scores = {
            unavailable: null,
            zero: 0,
            scored: 7.234,
        };
        const results = Object.entries(scores).map(([id, cvss]) => ({
            id,
            cve: `CVE-${id}`,
            cvss,
            scoreVersion: 'V3',
            isFixable: false,
            deploymentCount: 1,
            summary: 'Example vulnerability',
            imageCount: 1,
            lastScanned: null,
        }));
        cy.intercept('POST', graphqlUrl('mostCommonImageVulnerabilities'), {
            body: { data: { results } },
        });

        cy.mount(
            <ComponentTestProvider>
                <workflowStateContext.Provider value={new WorkflowState(useCases.VULN_MANAGEMENT)}>
                    <div style={{ width: 700, height: 400 }}>
                        <MostCommonVulnerabilities />
                    </div>
                </workflowStateContext.Provider>
            </ComponentTestProvider>
        );

        cy.contains('CVE-unavailable / CVSS: N/A (V3)').should('be.visible');
        cy.contains('CVE-zero / CVSS: 0.0 (V3)').should('be.visible');
        cy.contains('CVE-scored / CVSS: 7.2 (V3)').should('be.visible');
    });
});
