import * as api from '../../constants/apiEndpoints';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithRandomString,
} from '../../helpers/formHelpers';
import { visitIntegrationsUrl } from '../../helpers/integrations';
import { getTableRowActionButtonByName } from '../../helpers/tableHelpers';

function assertClusterInitBundleTable() {
    const label = labels.authProviders.clusterInitBundle;
    cy.get(`${selectors.breadcrumbItem}:contains("${label}")`);
    cy.get(`${selectors.title2}:contains("${label}")`);
}

const visitClusterInitBundlesUrl = `${url}/authProviders/clusterInitBundle`;
const createClusterInitBundleUrl = `${url}/authProviders/clusterInitBundle/create`;
const viewClusterInitBundleUrl = `${url}/authProviders/clusterInitBundle/view/`; // followed by id

function visitClusterInitBundles() {
    visitIntegrationsUrl(visitClusterInitBundlesUrl);
    assertClusterInitBundleTable();
}

describe('Cluster Init Bundle tests', () => {
    withAuth();

    // we have to use a randomstring here because using a name with a date is not a valid clusterInitBundle name
    const clusterInitBundleName = generateNameWithRandomString('ClusterInitBundleTest');

    it('should create a new Cluster Init Bundle integration', () => {
        visitClusterInitBundles();

        cy.get(selectors.buttons.newClusterInitBundle).click();
        cy.location('pathname').should('eq', createClusterInitBundleUrl);

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Cluster init bundle name').type(' ').blur();

        getHelperElementByLabel('Cluster init bundle name').contains(
            'A cluster init bundle name is required'
        );
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check fields for invalid formats
        getInputByLabel('Cluster init bundle name').type('Name with space/stuff').blur();

        getHelperElementByLabel('Cluster init bundle name').contains(
            'Name must contain only alphanumeric, ., _, or - (no spaces).'
        );
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 3, check valid from and generate
        getInputByLabel('Cluster init bundle name').clear().type(clusterInitBundleName);

        cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
        cy.intercept('POST', api.integration.clusterInitBundle.generate).as(
            'generateClusterInitBundle'
        );
        cy.get(selectors.buttons.generate).should('be.enabled').click();
        cy.wait(['@generateClusterInitBundle', '@getClusterInitBundles']);
        cy.get('[aria-label="Success Alert"]').should('contain', 'Download Helm values file');
        cy.get('[aria-label="Success Alert"]').should(
            'contain',
            'Download Kubernetes secrets file'
        );

        cy.get(selectors.buttons.back).click();
        assertClusterInitBundleTable();
    });

    it('should show the generated Cluster Init Bundle in the table, and be clickable', () => {
        visitClusterInitBundles();

        cy.get(`${selectors.tableRowNameLink}:contains("${clusterInitBundleName}")`).click();

        cy.location('pathname').should('contain', viewClusterInitBundleUrl);
        cy.get(`${selectors.breadcrumbItem}:contains("${clusterInitBundleName}")`);
    });

    it('should be able to revoke the Cluster Init Bundle', () => {
        visitClusterInitBundles();

        cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
        cy.intercept('PATCH', api.integration.clusterInitBundle.revoke).as(
            'revokeClusterInitBundle'
        );
        getTableRowActionButtonByName(clusterInitBundleName).click();
        cy.get('button:contains("Delete Integration")').click();
        cy.get(selectors.buttons.delete).click();
        cy.wait(['@revokeClusterInitBundle', '@getClusterInitBundles']);

        assertClusterInitBundleTable();
        cy.get(`${selectors.tableRowNameLink}:contains("${clusterInitBundleName}")`).should(
            'not.exist'
        );
    });
});
