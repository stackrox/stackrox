import withAuth from '../../helpers/basicAuth';

import { getInputByLabel } from '../../helpers/formHelpers';
import { visitWithStaticResponseForPermissions } from '../../helpers/visit';

import {
    visitClusters,
    visitDelegateScanning,
    saveDelegatedRegistryConfig,
    delegatedScanningPath,
    clustersPath,
} from './Clusters.helpers';

// There is some overlap between tests for Certificate Expiration and Health Status.
describe('Delegated Image Scanning', () => {
    withAuth();

    it(`should have a link on the clusters main page`, () => {
        visitClusters();

        cy.get('a:contains("Delegated image scanning")').click();

        cy.location('pathname').should('eq', '/main/clusters/delegated-image-scanning');
    });

    it(`should load the page in the cluster hierarchy`, () => {
        visitDelegateScanning();

        // make sure the static page loads
        cy.get('h1:contains("Delegated image scanning")');

        cy.get('.pf-v5-c-breadcrumb__item a:contains("Clusters")').should(
            'have.attr',
            'href',
            '/main/clusters'
        );

        cy.get('.pf-v5-c-breadcrumb__item:contains("Delegated image scanning")');

        // Apparently the initial state of central in CI
        // Delegate scanning for: Specified registries
        /*
        // check the initial state of the delegate config
        getInputByLabel('None').should('be.checked');

        cy.get('label:contains("All registries")').should('not.be.checked');
        cy.get('label:contains("Specified registries")').should('not.be.checked');
        */

        cy.get('button:contains("Edit")').click();

        // Enable delegate scanning with default
        getInputByLabel('All registries').click();

        getInputByLabel('All registries').should('be.checked');
        getInputByLabel('Specified registries').should('not.be.checked');

        // change the type of enabled for
        getInputByLabel('Specified registries').click();
        getInputByLabel('All registries').should('not.be.checked');
        getInputByLabel('Specified registries').should('be.checked');

        // Apparently the initial state of central in CI
        // Default cluster to delegate to: remote
        /*
        // None should be value for default cluster
        cy.get('[aria-label="Select default cluster"]')
            .should('have.text', 'None')
            .should('have.value', '');

        // choose the first cluster in the list as the default
        cy.get('[aria-label="Select default cluster"]').click();
        cy.get(
            '[aria-label="Select default cluster"] + .pf-v5-c-menu .pf-v5-c-menu__list-item'
        ).then(($clusterNames) => {
            expect($clusterNames.length).to.be.gte(0);
        });
        cy.get('[aria-label="Select default cluster"] + .pf-v5-c-menu .pf-v5-c-menu__list-item')
            .last()
            .then(($lastCluster) => {
                // Beware that in local deployment and some CI environments, None is only option.
                const lastClusterName = $lastCluster.text();
                cy.log('lastClusterName', lastClusterName);

                cy.wrap($lastCluster).click();

                cy.get('[aria-label="Select default cluster"]').should(
                    'have.text',
                    lastClusterName
                );

                // save the configuration
                saveDelegatedRegistryConfig();

                cy.get(
                    '.pf-v5-c-alert.pf-m-success .pf-v5-c-alert__title:contains("Delegated image scanning configuration saved successfully")'
                );
            });
        */
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
                cy.get('h1:contains("Delegated image scanning")').should('not.exist');
            });
        });

        it(`should not have a link on the Clusters page`, () => {
            cy.fixture('auth/mypermissionsNoAdminAccess.json').then(({ resourceToAccess }) => {
                const staticResponseForPermissions = {
                    body: {
                        resourceToAccess: { ...resourceToAccess },
                    },
                };

                visitWithStaticResponseForPermissions(clustersPath, staticResponseForPermissions);

                // make sure link is not present
                cy.get('a:contains("Delegated image scanning")').should('not.exist');
            });
        });
    });
});
