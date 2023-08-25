import withAuth from '../../helpers/basicAuth';
import {
    generateNameWithDate,
    getHelperElementByLabel,
    getInputByLabel,
} from '../../helpers/formHelpers';

import {
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';

// Page address segments are the source of truth for integrationSource and integrationType.
const integrationSource = 'signatureIntegrations';

describe('Signature Integrations', () => {
    withAuth();

    it('should create a new signature integration and then view and delete', () => {
        const integrationName = generateNameWithDate('Signature Integration');
        const integrationType = 'signature';

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

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(integrationSource, integrationType);

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

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View it.

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).click();

        cy.get(`${selectors.breadcrumbItem}:contains("${integrationName}")`);

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete it.

        deleteIntegrationInTable(integrationSource, integrationType, integrationName);

        cy.get(`${selectors.tableRowNameLink}:contains("${integrationName}")`).should('not.exist');
    });
});
