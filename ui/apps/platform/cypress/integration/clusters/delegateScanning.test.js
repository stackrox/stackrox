import withAuth from '../../helpers/basicAuth';

import { getInputByLabel } from '../../helpers/formHelpers';

import { visitDelegateScanning } from './Clusters.helpers';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Delegate Image Scanning', () => {
    withAuth();

    it(`should load the page in the cluster hierarchy`, () => {
        visitDelegateScanning();

        // make sure the static page loads
        cy.get('h1:contains("Delegate Image Scanning")');

        cy.get('.pf-c-breadcrumb__item a:contains("Clusters")').should(
            'have.attr',
            'href',
            '/main/clusters'
        );

        cy.get('.pf-c-breadcrumb__item:contains("Delegate Image Scanning")');

        // check the initial state of the delegate config
        getInputByLabel('Enable delegated image scanning').should('not.be.checked');

        cy.get('label:contains("All registries")').should('not.be.exist');
        cy.get('label:contains("Specified registries")').should('not.be.exist');

        // Enable delegate scanning with default
        getInputByLabel('Enable delegated image scanning').click();

        getInputByLabel('All registries').should('be.checked');
        getInputByLabel('Specified registries').should('not.be.checked');
    });
});
