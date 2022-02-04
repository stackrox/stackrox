import * as api from '../../constants/apiEndpoints';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import withAuth from '../../helpers/basicAuth';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateNameWithDate,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';
import { visitIntegrationsUrl } from '../../helpers/integrations';
import { getTableRowActionButtonByName } from '../../helpers/tableHelpers';

function assertAPITokenTable() {
    const label = labels.authProviders.apitoken;
    cy.get(`${selectors.breadcrumbItem}:contains("${label}")`);
    cy.get(`${selectors.title2}:contains("${label}")`);
}

const visitAPITokensUrl = `${url}/authProviders/apitoken`;
const createAPITokenUrl = `${url}/authProviders/apitoken/create`;
const viewAPITokenUrl = `${url}/authProviders/apitoken/view/`; // followed by id

function visitAPITokens() {
    visitIntegrationsUrl(visitAPITokensUrl);
    assertAPITokenTable();
}

describe('API Token tests', () => {
    withAuth();

    const apiTokenName = generateNameWithDate('API Token Test');

    it('should create a new API Token integration', () => {
        visitAPITokens();

        cy.intercept('GET', '/v1/roles').as('getRoles');
        cy.get(selectors.buttons.newApiToken).click();
        cy.wait('@getRoles');
        cy.location('pathname').should('eq', createAPITokenUrl);

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

        cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
        cy.intercept('POST', api.integration.apiToken.generate).as('generateAPIToken');
        cy.get(selectors.buttons.generate).should('be.enabled').click();
        cy.wait(['@generateAPIToken', '@getAPITokens']);
        cy.get('[aria-label="Success Alert"]');

        cy.get(selectors.buttons.back).click();
        assertAPITokenTable();
    });

    it('should show the generated API token in the table, and be clickable', () => {
        visitAPITokens();

        cy.intercept('GET', '/v1/roles').as('getRoles');
        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).click();
        cy.wait('@getRoles');

        cy.location('pathname').should('contain', viewAPITokenUrl);
        cy.get(`${selectors.breadcrumbItem}:contains("${apiTokenName}")`);
    });

    it('should be able to revoke the API token', () => {
        visitAPITokens();

        getTableRowActionButtonByName(apiTokenName).click();

        cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
        cy.intercept('PATCH', api.integration.apiToken.revoke).as('revokeAPIToken');
        cy.get('button:contains("Delete Integration")').click();
        cy.get(selectors.buttons.delete).click();
        cy.wait(['@revokeAPIToken', '@getAPITokens']);

        assertAPITokenTable();
        cy.get(`${selectors.tableRowNameLink}:contains("${apiTokenName}")`).should('not.exist');
    });
});
