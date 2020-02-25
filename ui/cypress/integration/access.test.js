import { selectors, url } from '../constants/AccessPage';
import * as api from '../constants/apiEndpoints';

import withAuth from '../helpers/basicAuth';

describe('Access Control Page', () => {
    describe('Auth Provider Rules', () => {
        withAuth();

        beforeEach(() => {
            cy.server();
            // TODO-ivan: remove once ROX_REFRESH_TOKENS is enabled by default
            cy.route('GET', api.featureFlags, {
                featureFlags: [
                    { name: 'Refresh tokens', envVar: 'ROX_REFRESH_TOKENS', enabled: true },
                    { name: 'Vuln Mgmt', envVar: 'ROX_VULN_MGMT_UI', enabled: false }
                ]
            }).as('featureFlags');

            cy.visit(url);
            cy.get(selectors.tabs.authProviders).click();
        });

        it('should open the new auth provider panel', () => {
            cy.get(selectors.authProviders.addProviderSelect).select(
                selectors.authProviders.newAuth0Option
            );
            cy.get(selectors.authProviders.newAuthProviderPanel).contains(
                'Create New Auth0 Auth Provider'
            );
        });

        it('should open the new OIDC provider form with client secret', () => {
            cy.get(selectors.authProviders.addProviderSelect).select(
                selectors.authProviders.newOidcOption
            );
            cy.get(selectors.authProviders.newAuthProviderPanel).contains(
                'Create New OpenID Connect Auth Provider'
            );

            // client secret should be marked as required as HTTP POST should be default callback
            cy.get(selectors.authProviders.clientSecretLabel).should(p => {
                expect(p.text()).to.contain('(required)');
            });
            cy.get(selectors.authProviders.doNotUseClientSecretCheckbox).should('not.be.disabled');
            cy.get(selectors.authProviders.clientSecretInput).should('not.be.disabled');

            // select Fragment
            cy.get(selectors.authProviders.fragmentCallbackRadio).check();

            // client secret fields will get disabled
            cy.get(selectors.authProviders.doNotUseClientSecretCheckbox).should('be.disabled');
            cy.get(selectors.authProviders.clientSecretInput).should('be.disabled');

            // select HTTP POST back
            cy.get(selectors.authProviders.httpPostCallbackRadio).check();

            // opt out from client secret usage
            cy.get(selectors.authProviders.doNotUseClientSecretCheckbox).check();
            cy.get(selectors.authProviders.clientSecretInput).should('be.disabled');
            cy.get(selectors.authProviders.clientSecretLabel).should(p => {
                expect(p.text()).not.to.contain('(required)');
            });
        });
    });

    describe('Roles and Permissions', () => {
        withAuth();

        beforeEach(() => {
            cy.visit(url);
            cy.get(selectors.tabs.roles).click();
        });

        const selectRole = roleName => {
            cy.get(selectors.roles)
                .contains(roleName)
                .click();
        };

        const createRole = roleName => {
            cy.get(selectors.addNewRoleButton).click();
            cy.get(selectors.permissionsPanelHeader).contains('Create New Role');
            cy.get(selectors.input.roleName).type(roleName);
            cy.get(selectors.saveButton).click();
            cy.get(selectors.roles)
                .contains(roleName)
                .then($role => {
                    cy.get(selectors.permissionsPanelHeader).contains(
                        `"${$role.text()}" Permissions`
                    );
                });
        };

        it('should have the default roles', () => {
            cy.get(selectors.roles).contains('Admin');
            cy.get(selectors.roles).contains('Analyst');
            cy.get(selectors.roles).contains('Continuous Integration');
            cy.get(selectors.roles).contains('None');
            cy.get(selectors.roles).contains('Sensor Creator');
        });

        it('should automatically select the first role', () => {
            cy.get(selectors.roles)
                .eq(0)
                .then($role => {
                    cy.get(selectors.permissionsPanelHeader).contains(
                        `"${$role.text()}" Permissions`
                    );
                });
        });

        it('should not be able to edit default roles', () => {
            selectRole('Admin');
            cy.get(selectors.editButton).should('not.exist');
            selectRole('Analyst');
            cy.get(selectors.editButton).should('not.exist');
            selectRole('Continuous Integration');
            cy.get(selectors.editButton).should('not.exist');
            selectRole('None');
            cy.get(selectors.editButton).should('not.exist');
            selectRole('Sensor Creator');
            cy.get(selectors.editButton).should('not.exist');
        });

        it('should create a new role', () => {
            const newRoleName = `Role-${new Date().getTime()}`;
            createRole(newRoleName);
        });

        it('should not be able to edit an existing role name', () => {
            const newRoleName = `Role-${new Date().getTime()}`;
            createRole(newRoleName);
            cy.get(selectors.editButton).click();
            cy.get(selectors.input.roleName).then($input => {
                cy.wrap($input).should('have.attr', 'disabled');
            });
        });
    });
});
