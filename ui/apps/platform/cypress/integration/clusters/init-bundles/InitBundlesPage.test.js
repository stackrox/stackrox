import withAuth from '../../../helpers/basicAuth';

import { interactAndVisitClusters, visitClusters } from '../Clusters.helpers';

import {
    assertInitBundlesPage,
    interactAndWaitForInitBundles,
    visitInitBundlesPage,
} from './InitBundles.helpers';

describe('Cluster init bundles InitBundlesPage', () => {
    withAuth();

    it('visits from clusters page', () => {
        visitClusters();

        interactAndWaitForInitBundles(() => {
            cy.get('a:contains("Init bundles")').click();
        });

        assertInitBundlesPage();
    });

    it('visits clusters from breadcrumb link', () => {
        visitInitBundlesPage();

        cy.get('.pf-v5-c-breadcrumb__item:nth-child(2):contains("Cluster init bundles")');
        interactAndVisitClusters(() => {
            cy.get('.pf-v5-c-breadcrumb__item:nth-child(1) a:contains("Clusters")').click();
        });
    });

    it('renders table head cells', () => {
        visitInitBundlesPage();

        cy.get('th:contains("Name")');
        cy.get('th:contains("Created by")');
        cy.get('th:contains("Created at")');
        cy.get('th:contains("Expires at")');
    });
});
