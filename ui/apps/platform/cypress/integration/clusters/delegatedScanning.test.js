import withAuth from '../../helpers/basicAuth';

import { getInputByLabel } from '../../helpers/formHelpers';
import { visitWithStaticResponseForPermissions } from '../../helpers/visit';

import {
    visitDelegateScanning,
    saveDelegatedRegistryConfig,
    delegatedScanningPath,
} from './Clusters.helpers';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Delegated Image Scanning', () => {
    withAuth();

    it(`should load the page in the cluster hierarchy`, () => {
        visitDelegateScanning();

        // make sure the static page loads
        cy.get('h1:contains("Delegated Image Scanning")');

        cy.get('.pf-c-breadcrumb__item a:contains("Clusters")').should(
            'have.attr',
            'href',
            '/main/clusters'
        );

        cy.get('.pf-c-breadcrumb__item:contains("Delegated Image Scanning")');

        // check the initial state of the delegate config
        getInputByLabel('Enable delegated image scanning').should('not.be.checked');

        cy.get('label:contains("All registries")').should('not.be.exist');
        cy.get('label:contains("Specified registries")').should('not.be.exist');

        // Enable delegate scanning with default
        getInputByLabel('Enable delegated image scanning').click();

        getInputByLabel('All registries').should('be.checked');
        getInputByLabel('Specified registries').should('not.be.checked');

        // change the type of enabled for
        getInputByLabel('Specified registries').click();
        getInputByLabel('All registries').should('not.be.checked');
        getInputByLabel('Specified registries').should('be.checked');

        // choose the first cluster in the list as the default
        cy.get('.cluster-select').click();
        cy.get('.cluster-select .pf-c-select__menu .pf-c-select__menu-item').then(
            ($clusterNames) => {
                expect($clusterNames.length).to.be.gte(0);
            }
        );
        cy.get('.cluster-select .pf-c-select__menu .pf-c-select__menu-item')
            .first()
            .then(($firstCluster) => {
                const firstClusterName = $firstCluster.text();

                $firstCluster.click();

                cy.get('.cluster-select').should('have.text', firstClusterName);
            });

        // save the configuration
        saveDelegatedRegistryConfig();

        cy.get(
            '.pf-c-alert.pf-m-success .pf-c-alert__title:contains("Delegated scanning configuration saved successfully")'
        );
    });

    describe('when user does not have permission to see page', () => {
        it(`should not show the page`, () => {
            cy.fixture('auth/mypermissionsNoAdminAccess.json').then(({ resourceToAccess }) => {
                const staticResponseForPermissions = {
                    body: {
                        resourceToAccess: { ...resourceToAccess },
                    },
                };

                visitWithStaticResponseForPermissions(
                    delegatedScanningPath,
                    staticResponseForPermissions
                );

                // make sure page does not load
                cy.get('h1:contains("Delegated Image Scanning")').should('not.exist');
            });
        });
    });
});
