import withAuth from '../../helpers/basicAuth';

import { visitDelegateScanning } from './Clusters.helpers';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Delegate Image Scanning', () => {
    withAuth();

    it(`should be a page in the cluster hierarchy`, () => {
        visitDelegateScanning();

        cy.get('h1:contains("Delegate Image Scanning")');

        cy.get('.pf-c-breadcrumb__item a:contains("Clusters")').should(
            'have.attr',
            'href',
            '/main/clusters'
        );

        cy.get('.pf-c-breadcrumb__item:contains("Delegate Image Scanning")');
    });
});
