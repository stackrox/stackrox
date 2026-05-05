import { createStore } from 'redux';

import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import ApiClientIntegrationsTab from './ApiClientIntegrationsTab';

const sourcesEnabled = ['apiClients'];

// OcmDeprecatedToken inside IntegrationsTabPage reads from Redux
const reduxStore = createStore(() => ({
    app: {
        cloudSources: {
            cloudSources: [],
        },
    },
}));

function setup() {
    cy.mount(
        <ComponentTestProvider reduxStore={reduxStore}>
            <ApiClientIntegrationsTab sourcesEnabled={sourcesEnabled} />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render the ServiceNow VR tile with the correct label', () => {
        setup();

        cy.findByText('ServiceNow VR').should('exist');
    });

    it('should display a description of the connector', () => {
        setup();

        cy.findByText('Pull RHACS vulnerability data into ServiceNow Vulnerability Response.').should(
            'exist'
        );
    });

    it('should link to the ServiceNow Store listing', () => {
        setup();

        const expectedUrl =
            'https://store.servicenow.com/store/app/edea7344476072502ec7c1c4f16d4343';

        cy.findByRole('link', {
            name: 'View ServiceNow VR app in ServiceNow Store (opens in a new tab)',
        })
            .should('have.attr', 'href', expectedUrl)
            .should('have.attr', 'target', '_blank');
    });

    it('should display the external link icon', () => {
        setup();

        cy.get('[data-testid="external-link-icon"]').should('exist');
    });
});
