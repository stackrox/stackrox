import withAuth from '../../helpers/basicAuth';
import { visitIntegrationsUrl } from '../../helpers/integrations';
import { labels, selectors, url } from '../../constants/IntegrationsPage';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';
import * as api from '../../constants/apiEndpoints';
import { getTableRowActionButtonByName } from '../../helpers/tableHelpers';

const visitSignatureIntegrationsUrl = `${url}/signatureIntegrations/signature`;

function assertSignatureIntegrationTable() {
    const label = labels.signatureIntegrations.signature;
    cy.get(`${selectors.breadcrumbItem}:contains("${label}")`);
}

function visitSignatureIntegrations() {
    visitIntegrationsUrl(visitSignatureIntegrationsUrl);
    assertSignatureIntegrationTable();
}

function saveSignatureIntegration() {
    cy.intercept('GET', api.integrations.signatureIntegrations).as('getSignatureIntegrations');
    // Mock request.
    cy.intercept('POST', api.integrations.signatureIntegrations).as('postSignatureIntegration');
    cy.get(selectors.buttons.save).should('be.enabled').click();
    cy.wait(['@postSignatureIntegration', '@getSignatureIntegrations']);
    assertSignatureIntegrationTable();
    cy.location('pathname').should('eq', visitSignatureIntegrationsUrl);
}

describe('Signature Integrations Test', () => {
    withAuth();

    const integrationName = generateNameWithDate('Signature Integration');

    it('should create a new signature integration', () => {
        const publicKeyValue =
            '-----BEGIN PUBLIC KEY-----\n' +
            'MIIBigKCAYEAnLceC91dTu1Lj6pMcLL3zcmps+NkczJPIaHDn8OtEnj+XzdmsMjO\n' +
            'zzmROtVH1HnsvDn5/tlxfqCMbWY1E6ezuj8wY9GY6eGHvEjU8JdZUw0Zoy2m3OV0\n' +
            'L3PDEuzATyT0fUjUNgjSXLNLLNl2LEF9yw/UP7QiHhj1mLojGUjaQ1REzBqkfsP2\n' +
            '7vR4AQbbf77/b5dwisoDYZXa+RnJ8IHWtXlnkBbf8eTo+8EArMGexpznSC4F5aL+\n' +
            '3aPl3Y2MFdmW2rDvjy4gNQQtBquJDIoyZEMTlDbMH4WV+44fZZfw0AP5MGPj1y+h\n' +
            'I1ea2UeFSkCWz+BDGHCj0kIUwLcDZaZfT4lu5qNe6XuEeTpPjnrEbqPf3NGg0DLQ\n' +
            'ZSpZ6ih3oWto2uTknM1Tf97Nr41J6nqec6Auott3oE9ww5KiJEiVi9q9L7cMupmS\n' +
            'xPP9jtUUiPdAw4uL71gLncP/YRYYyvjH3/aveFSlc83mS808FTRHiNfwBKHppuLW\n' +
            'HS1I6y+PPPrVAgMBAAE=\n' +
            '-----END PUBLIC KEY-----\n';
        visitSignatureIntegrations();
        cy.get(selectors.buttons.newIntegration).click();

        // Check inital state.
        cy.get(selectors.buttons.save).should('be.disabled');

        // Check empty values are not accepted.
        getInputByLabel('Integration name').type(' ');
        cy.get('button:contains("Cosign")').click({ force: true });
        cy.get('button:contains("Add new public key")').click({ force: true });
        getInputByLabel('Public key name').type('  ');
        getInputByLabel('Public key value').type('  ');

        getHelperElementByLabel('Integration name').contains('Integration name is required');
        getHelperElementByLabel('Public key name').contains('Name is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Save integration.
        getInputByLabel('Integration name').clear().type(integrationName);
        getInputByLabel('Public key name').clear().type('keyName');
        getInputByLabel('Public key value').clear().type(publicKeyValue);
        cy.get(selectors.buttons.save).should('be.enabled');
        saveSignatureIntegration();
    });

    it('should show created signature integration in the table, and be clickable', () => {
        visitSignatureIntegrations();

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();
        cy.location('pathname').should('contain', visitSignatureIntegrationsUrl);
        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);
    });

    it('should be able to delete the signature integration', () => {
        visitSignatureIntegrations();

        getTableRowActionButtonByName(integrationName).click();

        cy.intercept('GET', api.integrations.signatureIntegrations).as('getSignatureIntegrations');
        cy.intercept('DELETE', `${api.integrations.signatureIntegrations}/*`).as(
            'deleteSignatureIntegration'
        );
        cy.get('button:contains("Delete Integration")').click();
        cy.get(selectors.buttons.delete).click();
        cy.wait(['@deleteSignatureIntegration', '@getSignatureIntegrations']);

        assertSignatureIntegrationTable();
        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });
});
