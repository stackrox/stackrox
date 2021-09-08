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

function getGeneratedClusterInitBundle(name) {
    return {
        meta: {
            id: 'f8694cb5-66bf-4e82-a03a-6665ae556321',
            name,
            impactedClusters: [],
            createdAt: '2021-09-03T19:03:16.794525437Z',
            createdBy: {
                id: 'sso:fbe77f87-6664-47e7-a84b-96e681aab2f5:google-oauth2|114513146969073096269',
                authProviderId: 'fbe77f87-6664-47e7-a84b-96e681aab2f5',
                attributes: [
                    {
                        key: 'email',
                        value: 'sc@stackrox.com',
                    },
                    {
                        key: 'name',
                        value: 'Saif Chaudhry',
                    },
                    {
                        key: 'userid',
                        value: 'google-oauth2|114513146969073096269',
                    },
                ],
            },
            expiresAt: '2022-09-03T19:03:00Z',
        },
        helmValuesBundle: 'IyBUaGlzIGlzIGEgU3RhY2tSb3ggY2x1c3RlciBpbml0IG',
        kubectlBundle: 'IyBUaGlzIGlzIGEgU3RhY2tSb3ggY2x1c3RlciBp',
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
        cy.intercept('POST', api.integration.apiToken.generate, generatedToken).as(
            'generateAPIToken'
        );

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

    it('should create a new Cluster Init Bundle integration', () => {
        const clusterInitBundleName = generateUniqueName('Cluster Init Bundle Test');
        const generatedClusterInitBundle = getGeneratedClusterInitBundle(clusterInitBundleName);
        cy.intercept(
            'POST',
            api.integration.clusterInitBundle.generate,
            generatedClusterInitBundle
        ).as('generateClusterInitBundle');

        cy.get(selectors.clusterInitBundleTile).click();

        // @TODO: only use the click, and delete the direct URL visit after forms official launch
        cy.get(selectors.buttons.new).click();
        cy.visit('/main/integrations/authProviders/clusterInitBundle/create');

        // Step 0, should start out with disabled Generate button
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 1, check empty fields
        getInputByLabel('Cluster init bundle name').type(' ').blur();

        getHelperElementByLabel('Cluster init bundle name').contains(
            'A cluster init bundle name is required'
        );
        cy.get(selectors.buttons.generate).should('be.disabled');

        // Step 2, check valid from and generate
        getInputByLabel('Cluster init bundle name').clear().type(clusterInitBundleName);

        cy.get(selectors.buttons.generate).should('be.enabled').click();
        cy.wait('@generateClusterInitBundle');

        cy.location().should((loc) => {
            expect(loc.pathname).to.eq('/main/integrations/authProviders/clusterInitBundle/create');
        });
        cy.get('[aria-label="Success Alert"]').should('contain', 'Download Helm values file');
        cy.get('[aria-label="Success Alert"]').should(
            'contain',
            'Download Kubernetes secrets file'
        );
    });
});
