import withAuth from '../../../helpers/basicAuth';
import { generateNameWithDate, getInputByLabel } from '../../../helpers/formHelpers';

import { interactAndVisitClusters } from '../Clusters.helpers';

import {
    assertInitBundleForm,
    assertInitBundlePage,
    assertInitBundlesPage,
    interactAndWaitForCreateBundle,
    interactAndWaitForInitBundles,
    interactAndWaitForRevokeBundle,
    visitInitBundleForm,
    visitInitBundlesPage,
} from './InitBundles.helpers';

describe('Cluster init bundles InitBundlesForm', () => {
    withAuth();

    it('visits clusters from breadcrumb link', () => {
        visitInitBundleForm();

        cy.get('.pf-v5-c-breadcrumb__item:nth-child(3):contains("Create bundle")');
        interactAndVisitClusters(() => {
            cy.get('.pf-v5-c-breadcrumb__item:nth-child(1) a:contains("Clusters")').click();
        });
    });

    it('visits cluster init bundles from breadcrumb link', () => {
        visitInitBundleForm();

        cy.get('.pf-v5-c-breadcrumb__item:nth-child(3):contains("Create bundle")');
        interactAndWaitForInitBundles(() => {
            cy.get(
                '.pf-v5-c-breadcrumb__item:nth-child(2) a:contains("Cluster init bundles")'
            ).click();
        });

        assertInitBundlesPage();
    });

    it('creates, views, and then deletes a bundle', () => {
        visitInitBundlesPage();

        const name = generateNameWithDate('Create-bundle').replace(/:/g, ''); // colon in not valid in name

        cy.get(`td[data-label="Name"] a:contains("${name}")`).should('not.exist');
        cy.get('a:contains("Create bundle")').click();

        assertInitBundleForm();
        cy.get('button:contains("Download")').should('be.disabled');
        getInputByLabel('Name').type(name);
        interactAndWaitForCreateBundle(() => {
            cy.get('button:contains("Download")').click();
        });

        assertInitBundlesPage();
        interactAndWaitForInitBundles(() => {
            cy.get(`td[data-label="Name"] a:contains("${name}")`).click();
        });

        assertInitBundlePage();
        cy.get('button:contains("Revoke bundle")').click();
        cy.get('.pf-v5-c-modal-box__title:contains("Revoke cluster init bundle")');
        interactAndWaitForRevokeBundle(() => {
            cy.get(`[role="dialog"] button:contains("Revoke bundle")`).click();
        });

        assertInitBundlesPage();
        cy.get(`td[data-label="Name"] a:contains("${name}")`).should('not.exist');
    });
});
