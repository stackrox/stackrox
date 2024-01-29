import { selectors, url } from '../constants/AccessPage';

import withAuth from '../helpers/basicAuth';

describe.skip('Access Control Page', () => {
    withAuth();

    describe('Auth Provider Rules', () => {
        function navigateToThePanel(authProviders = 'fixture:auth/authProviders-id1-id2-id3.json') {
            cy.server();
            cy.route('GET', '/v1/authProviders', authProviders).as('authProviders');

            cy.visit(url);
            cy.get(selectors.tabs.authProviders).click();
            cy.wait('@authProviders');
        }

        it('should open the new auth provider panel', () => {
            navigateToThePanel();

            cy.get(selectors.authProviders.addProviderSelect.input).click();
            cy.get(
                `${selectors.authProviders.addProviderSelect.options}:contains("${selectors.authProviders.newAuth0Option}")`
            ).click();
            cy.get(selectors.authProviders.authProviderPanel).contains(
                'Create New Auth0 Auth Provider'
            );
        });

        it('should open the new OIDC provider form with client secret', () => {
            navigateToThePanel();

            cy.get(selectors.authProviders.addProviderSelect.input).click();
            cy.get(
                `${selectors.authProviders.addProviderSelect.options}:contains("${selectors.authProviders.newOidcOption}")`
            ).click();
            cy.get(selectors.authProviders.authProviderPanel).contains(
                'Create New OpenID Connect Auth Provider'
            );

            // client secret should be marked as required as Auto-select should be default callback
            cy.get(selectors.authProviders.clientSecretLabel).should((p) => {
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
            cy.get(selectors.authProviders.clientSecretLabel).should((p) => {
                expect(p.text()).not.to.contain('(required)');
            });
        });

        it('should modify an auth provider with a client secret without losing the value', () => {
            navigateToThePanel('fixture:auth/authProvidersWithClientSecret.json');
            cy.server();
            cy.route('PUT', '/v1/authProviders/auth-provider-1', {}).as('saveAuthProvider');

            cy.get(`${selectors.authProviderDetails.clientSecret}:contains("*****")`);

            cy.get(selectors.editButton).click();

            cy.get(selectors.authProviders.doNotUseClientSecretCheckbox).should('not.be.disabled');
            cy.get(selectors.authProviders.clientSecretInput).should('not.be.disabled');

            cy.get(selectors.input.issuer).clear();
            cy.get(selectors.input.issuer).type('irrelevant-updated');
            cy.get(selectors.saveButton).click();

            cy.get(`${selectors.authProviderDetails.clientSecret}:contains("*****")`);
        });

        it('should select the first auth provider by default', () => {
            navigateToThePanel();

            const { leftSidePanel } = selectors.authProviders;
            cy.get(leftSidePanel.selectedRow).contains('auth-provider-1');
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('auth-provider-1');
        });

        it('should select the first auth provider after deleting the selected one', () => {
            navigateToThePanel();
            const { leftSidePanel } = selectors.authProviders;

            cy.get(leftSidePanel.secondRow).click({ force: true }); // forcing since it's a div
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('auth-provider-2'); // check selection

            cy.get(leftSidePanel.selectedRowDeleteButton).click({ force: true }); // forcing as its a hover button

            // mock now with the second one deleted
            cy.route('GET', '/v1/authProviders', 'fixture:auth/authProviders-id1-id3.json').as(
                'authProviders'
            );

            cy.get(selectors.modal.deleteButton).click();

            cy.wait('@authProviders');
            // first one should become selected
            cy.get(leftSidePanel.selectedRow).contains('auth-provider-1');
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('auth-provider-1');
        });

        it('should not change selection if non-selected deleted', () => {
            navigateToThePanel();
            const { leftSidePanel } = selectors.authProviders;

            cy.get(leftSidePanel.thirdRow).click({ force: true }); // forcing since it's a div
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('auth-provider-3'); // check selection

            cy.get(leftSidePanel.secondRowDeleteButton).click({ force: true }); // forcing as its a hover button

            // mock now with the second one deleted
            cy.route('GET', '/v1/authProviders', 'fixture:auth/authProviders-id1-id3.json').as(
                'authProviders'
            );

            cy.get(selectors.modal.deleteButton).click();

            cy.wait('@authProviders');
            // third one should remain selected
            cy.get(leftSidePanel.selectedRow).contains('auth-provider-3');
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('auth-provider-3');
        });

        it('should show empty state after deleting the last provider', () => {
            navigateToThePanel();
            const { leftSidePanel } = selectors.authProviders;

            cy.get(leftSidePanel.selectedRowDeleteButton).click({ force: true }); // forcing as its a hover button

            // mock now with empty list of providers like nothing is left
            cy.route('GET', '/v1/authProviders', { authProviders: [] }).as('authProviders');
            cy.get(selectors.modal.deleteButton).click();

            cy.wait('@authProviders');
            // should show empty state
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('Getting Started');
        });

        it('should show empty state with no auth providers', () => {
            navigateToThePanel({ authProviders: [] });
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('Getting Started');
        });

        it('should return to the empty state after cancelling addition of the first provider (ROX-4359)', () => {
            navigateToThePanel({ authProviders: [] });

            cy.get(selectors.authProviders.addProviderSelect.input).click();
            cy.get(
                `${selectors.authProviders.addProviderSelect.options}:contains("${selectors.authProviders.newOidcOption}")`
            ).click();
            cy.get(selectors.authProviders.authProviderPanel).contains(
                'Create New OpenID Connect Auth Provider'
            );

            cy.get(selectors.cancelButton).click();
            cy.get(selectors.authProviders.authProviderPanelHeader).contains('Getting Started');
        });
    });

    describe('Roles and Permissions', () => {
        beforeEach(() => {
            cy.visit(url);
            cy.get(selectors.tabs.roles).click();
        });

        const selectRole = (roleName) => {
            cy.get(selectors.roles).contains(roleName).click();
        };

        const createRole = (roleName) => {
            cy.get(selectors.addNewRoleButton).click();
            cy.get(selectors.permissionsPanelHeader).contains('Create New Role');
            cy.get(selectors.input.roleName).type(roleName);
            cy.get(selectors.saveButton).click();
            cy.get(selectors.roles)
                .contains(roleName)
                .then(($role) => {
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
                .then(($role) => {
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
            cy.get(selectors.input.roleName).then(($input) => {
                cy.wrap($input).should('have.attr', 'disabled');
            });
        });

        it('should have Alert permission', () => {
            cy.get(selectors.permissionsMatrix.rowByPermission('Alert')).should('exist');
        });
    });

    describe('with limited permissions', () => {
        beforeEach(() => {
            cy.server();

            cy.fixture('auth/mypermissionsMinimalAccess.json').as('minimalPermissions');

            cy.route('GET', 'v1/mypermissions', '@minimalPermissions').as('getMyPermissions');

            cy.visit(url);
            cy.wait('@getMyPermissions');
        });

        it('should show no access instead of auth providers', () => {
            cy.get(selectors.message).contains(
                'You do not have permission to view Access Control.'
            );
        });
    });
});
