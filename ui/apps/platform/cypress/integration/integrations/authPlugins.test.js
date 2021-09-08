import { selectors } from '../../constants/IntegrationsPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { editIntegration } from './integrationUtils';
import {
    getHelperElementByLabel,
    getInputByLabel,
    generateUniqueName,
    getSelectButtonByLabel,
    getSelectOption,
} from '../../helpers/formHelpers';

function getGeneratedAPIToken(name) {
    return {
        // shortened for readability
        token: 'eyJhbGciOiJSUzI1NiIsImtpZCI',
        metadata: {
            id: '41e329a5-a752-43ac-91ff-3dfd82d7ab1d',
            name,
            roles: ['Admin'],
            issuedAt: '2021-09-03T17:58:33Z',
            expiration: '2022-09-03T17:58:33Z',
            revoked: false,
            role: 'Admin',
        },
    };
}

describe('Authorization Plugin Test', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', api.integrations.authPlugins, {
            fixture: 'integrations/authPlugins.json',
        }).as('getAuthPlugins');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getAuthPlugins');
    });

    it('should show a hint about stored credentials for Scoped Access Plugin', () => {
        cy.get(selectors.scopedAccessPluginTile).click();
        editIntegration('Scoped Access Plugin Test');
        cy.get('div:contains("Password"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
        cy.get('div:contains("Client Key"):last [data-testid="help-icon"]').trigger('mouseenter');
        cy.get(selectors.tooltip.overlay).contains(
            'Leave this empty to use the currently stored credentials'
        );
    });
});

describe('Authentication Plugins Forms', () => {
    withAuth();

    beforeEach(() => {
        cy.intercept('GET', api.integrations.apiTokens).as('getAPITokens');
        cy.intercept('GET', api.integrations.clusterInitBundles).as('getClusterInitBundles');

        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click({ force: true });
        cy.wait('@getAPITokens');
        cy.wait('@getClusterInitBundles');
    });

    it('should create a new API Token integration', () => {
        const apiTokenName = generateUniqueName('API Token Test');
        const generatedToken = getGeneratedAPIToken(apiTokenName);
        cy.intercept('POST', api.apiToken.generate, generatedToken).as('generateAPIToken');

        cy.get(selectors.apiTokenTile).click();

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/authProviders/apitoken/create');

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

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/authProviders/apitoken/create');
        });
        cy.get('[aria-label="Success Alert"]').should('contain', generatedToken.token);
    });
});
