import { selectors, url as integrationsUrl } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithRandomString,
} from '../../helpers/formHelpers';
import { getTableRowActionButtonByName, getTableRowLinkByName } from '../../helpers/tableHelpers';

describe('Cluster Init Bundle tests', () => {
    withAuth();

    // we have to use a randomstring here because using a name with a date is not a valid clusterInitBundle name
    const clusterInitBundleName = generateNameWithRandomString('ClusterInitBundleTest');

    beforeEach(() => {
        cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');
        cy.intercept('POST', api.integration.clusterInitBundle.generate).as(
            'generateClusterInitBundle'
        );
        cy.intercept('PATCH', api.integration.clusterInitBundle.revoke).as(
            'revokeClusterInitBundle'
        );

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getClusterInitBundles');
    });

    it('should create a new Cluster Init Bundle integration', () => {
        cy.get(selectors.clusterInitBundleTile).click();

        cy.get(selectors.buttons.newClusterInitBundle).click();

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

        cy.get(selectors.buttons.generate).should('be.enabled').click();
        cy.wait('@generateClusterInitBundle');

        cy.location('pathname').should(
            'eq',
            `${integrationsUrl}/authProviders/clusterInitBundle/create`
        );
        cy.get('[aria-label="Success Alert"]').should('contain', 'Download Helm values file');
        cy.get('[aria-label="Success Alert"]').should(
            'contain',
            'Download Kubernetes secrets file'
        );
    });

    it('should show the generated Cluster Init Bundle in the table, and be clickable', () => {
        cy.get(selectors.clusterInitBundleTile).click();

        getTableRowLinkByName(clusterInitBundleName).click();

        cy.location('pathname').should('contain', 'view');
    });

    it('should be able to revoke the Cluster Init Bundle', () => {
        cy.get(selectors.clusterInitBundleTile).click();

        getTableRowActionButtonByName(clusterInitBundleName).click();
        cy.get('button:contains("Delete Integration")').click();
        cy.get('button:contains("Delete")').click();
        cy.wait('@revokeClusterInitBundle');

        getTableRowActionButtonByName(clusterInitBundleName).should('not.exist');
    });
});
