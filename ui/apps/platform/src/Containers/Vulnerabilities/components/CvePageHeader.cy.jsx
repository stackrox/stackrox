import React from 'react';
import { createStore } from 'redux';

import ComponentTestProviders from 'test-utils/ComponentProviders';

import CvePageHeader from './CvePageHeader';

const readOnlyReduxStore = createStore((s) => s, {
    app: {
        featureFlags: {
            featureFlags: [],
            loading: false,
            error: null,
        },
    },
});
function setup(data) {
    cy.intercept('GET', '/v1/featureFlags', (req) => {
        req.reply({ data: { featureFlags: [] } });
    });

    cy.mount(
        <ComponentTestProviders reduxStore={readOnlyReduxStore}>
            <CvePageHeader data={data} />
        </ComponentTestProviders>
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
