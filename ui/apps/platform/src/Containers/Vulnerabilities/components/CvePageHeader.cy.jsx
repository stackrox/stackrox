import React from 'react';

import { FeatureFlagsProvider } from 'providers/FeatureFlagProvider';
import CvePageHeader from './CvePageHeader';

function setup(data) {
    cy.intercept('GET', '/v1/featureFlags', (req) => {
        req.reply({ data: { featureFlags: [] } });
    });

    cy.mount(
        <FeatureFlagsProvider>
            <CvePageHeader data={data} />
        </FeatureFlagsProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render loading skeletons when data is `undefined`', () => {
        setup(undefined);

        cy.get('h1').should('not.exist');
        cy.findByText('Loading CVE name');
        cy.findByText('Loading CVE metadata');
    });

    it('should not render empty elements when data is missing', () => {
        setup({ cve: 'CVE-2021-1234', firstDiscoveredInSystem: undefined, distroTuples: [] });

        // No distros, no link
        cy.findByRole('link').should('not.exist');
        // firstDiscoveredInSystem is undefined, so do not show labels
        cy.get('.pf-c-label-group').should('not.exist');
        cy.get('.pf-c-label').should('not.exist');
    });
});
