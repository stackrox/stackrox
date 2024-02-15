import withAuth from '../../helpers/basicAuth';
import {
    clickCreateNewIntegrationInTable,
    clickIntegrationSourceLinkInForm,
    deleteIntegrationInTable,
    saveCreatedIntegrationInForm,
    visitIntegrationsTable,
} from './integrations.helpers';
import { selectors } from './integrations.selectors';
import {
    getHelperElementByLabel,
    getInputByLabel,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';
import { hasFeatureFlag } from '../../helpers/features';

const integrationSource = 'authProviders';

describe('Machine Access Configs', () => {
    withAuth();
    
    before(function () {
        if (!hasFeatureFlag('ROX_AUTH_MACHINE_TO_MACHINE')) {
            this.skip();
        }
    });


    it('should create a new Machine Access integration and then view and delete', () => {
        const integrationType = 'machineAccess';

        visitIntegrationsTable(integrationSource, integrationType);
        clickCreateNewIntegrationInTable(
            integrationSource,
            integrationType,
            'Create configuration',
            'Create configuration'
        );

        // Check inital state.
        cy.get(selectors.buttons.save).should('be.disabled');

        // Check that issuer is automatically determined when Github action type is selected.
        getSelectButtonByLabel('Select configuration type').click();
        getSelectOption('Github action').click();
        getInputByLabel('Issuer').should('be.disabled');
        getInputByLabel('Issuer').should(
            'contain.value',
            'https://token.actions.githubusercontent.com'
        );

        // Check that empty values are not accepted.
        getSelectButtonByLabel('Select configuration type').click();
        getSelectOption('Generic').click();
        getInputByLabel('Issuer').clear().type(' ');
        getInputByLabel('Token lifetime').clear().type(' ').blur();

        getHelperElementByLabel('Issuer').contains('Issuer is required');
        getHelperElementByLabel('Token lifetime').contains('Token lifetime is required');
        cy.get(selectors.buttons.save).should('be.disabled');

        // Save integration.
        getSelectButtonByLabel('Select configuration type').click();
        getSelectOption('Github action').click();
        getInputByLabel('Token lifetime').clear().type('3h20m');

        // Check that without rules it's not possible to save integration.
        cy.get(selectors.buttons.save).should('be.disabled');

        cy.get('button:contains("Add new rule")').click();

        getInputByLabel('Key').clear().type('  ');
        getInputByLabel('Value').clear().type('  ').blur();

        // Check that empty rule is not accepted.
        getHelperElementByLabel('Key').contains('Key is required');
        getHelperElementByLabel('Value').contains('Value expression is required');

        getInputByLabel('Key').clear().type('key');
        getInputByLabel('Value').clear().type('value');
        getSelectButtonByLabel('Role').click();
        getSelectOption('Admin').click();

        saveCreatedIntegrationInForm(integrationSource, integrationType);

        // View it.
        cy.get(`${selectors.tableRowNameLink}:contains("Github action")`).click();

        clickIntegrationSourceLinkInForm(integrationSource, integrationType);

        // Delete it.
        deleteIntegrationInTable(integrationSource, integrationType, 'Github action');

        cy.get(`${selectors.tableRowNameLink}:contains("Github action")`).should('not.exist');
    });
});
