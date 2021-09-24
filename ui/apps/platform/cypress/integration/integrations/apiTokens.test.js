import { selectors, url as integrationsUrl } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';
import { getTableRowActionButtonByName, getTableRowLinkByName } from '../../helpers/tableHelpers';

describe('API Token tests', () => {
    withAuth();

    const apiTokenName = generateNameWithDate('API Token Test');

    beforeEach(() => {
        cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
        cy.intercept('POST', api.integration.apiToken.generate).as('generateAPIToken');
        cy.intercept('PATCH', api.integration.apiToken.revoke).as('revokeAPIToken');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getAPITokens');
    });

    it('should create a new API Token integration', () => {
        cy.get(selectors.apiTokenTile).click();

        cy.get(selectors.buttons.newApiToken).click();

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Token name').type(' ').blur();

        getHelperElementByLabel('Token name').contains('A token name is required');
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check valid from and generate
        getInputByLabel('Token name').clear().type(apiTokenName);
        getSelectButtonByLabel('Role').click();
        getSelectOption('Admin').click();

        cy.get(selectors.buttons.generate).should('be.enabled').click();
        cy.wait('@generateAPIToken');

        cy.location('pathname').should('eq', `${integrationsUrl}/authProviders/apitoken/create`);
        cy.get('[aria-label="Success Alert"]');
    });

    it('should show the generated API token in the table, and be clickable', () => {
        cy.get(selectors.apiTokenTile).click();

        getTableRowLinkByName(apiTokenName).click();

        cy.location('pathname').should('contain', 'view');
    });

    it('should be able to revoke the API token', () => {
        cy.get(selectors.apiTokenTile).click();

        getTableRowActionButtonByName(apiTokenName).click();
        cy.get('button:contains("Delete Integration")').click();
        cy.get(selectors.buttons.delete).click();
        cy.wait('@revokeAPIToken');

        getTableRowActionButtonByName(apiTokenName).should('not.exist');
    });
});
