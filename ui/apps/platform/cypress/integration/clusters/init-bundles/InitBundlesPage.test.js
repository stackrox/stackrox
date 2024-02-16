import withAuth from '../../../helpers/basicAuth';
import { hasFeatureFlag } from '../../../helpers/features';

import { interactAndVisitClusters, visitClusters } from '../Clusters.helpers';

import {
    assertInitBundlesPage,
    interactAndWaitForInitBundles,
    visitInitBundlesPage,
} from './InitBundles.helpers';

describe('Cluster init bundles InitBundlesPage', () => {
    withAuth();

    before(function () {
        if (!hasFeatureFlag('ROX_MOVE_INIT_BUNDLES_UI')) {
            this.skip();
        }
    });

    it('visits from clusters page', () => {
        visitClusters();

        interactAndWaitForInitBundles(() => {
            cy.get('a:contains("Init bundles")').click();
        });

        assertInitBundlesPage();
    });

    it('visits clusters from breadcrumb link', () => {
        visitInitBundlesPage();

        cy.get('.pf-c-breadcrumb__item:nth-child(2):contains("Cluster init bundles")');
        interactAndVisitClusters(() => {
            cy.get('.pf-c-breadcrumb__item:nth-child(1) a:contains("Clusters")').click();
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
