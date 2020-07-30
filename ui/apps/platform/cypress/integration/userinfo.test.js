import checkFeatureFlag from '../helpers/features';
import withAuth from '../helpers/basicAuth';
import { selectors as userPageSelectors, url as userPageUrl } from '../constants/UserPage';
import { url as dashboardURL } from '../constants/DashboardPage';
import { selectors as topNavSelectors } from '../constants/TopNavigation';
import * as api from '../constants/apiEndpoints';

describe('User Info', () => {
    withAuth();

    before(function beforeHook() {
        // skip the whole suite if user info feature isn't enabled
        if (checkFeatureFlag('ROX_CURRENT_USER_INFO', false)) {
            this.skip();
        }
    });

    function mockWithAdminUser() {
        cy.server();
        cy.route('GET', api.auth.authStatus, 'fixture:auth/adminUserStatus').as('authStatus');
    }

    function mockWithMultiRolesUser() {
        cy.server();
        cy.route('GET', api.auth.authStatus, 'fixture:auth/multiRolesUserStatus').as('authStatus');
    }

    function mockWithBasicAuthUser() {
        cy.server();
        cy.route('GET', api.auth.authStatus, 'fixture:auth/basicAuthAdminStatus').as('authStatus');
    }

    describe('User Info in Top Navigation', () => {
        it('should show initials in the user avatar', () => {
            mockWithAdminUser();
            cy.visit(dashboardURL);
            cy.get(topNavSelectors.menuButton).should('contain.text', 'AI');
        });

        it('should show name, email and a single role', () => {
            mockWithAdminUser();
            cy.visit(dashboardURL);
            cy.get(topNavSelectors.menuButton).click();

            cy.get(topNavSelectors.menuList.userName).should(
                'contain.text',
                'Artificial Intelligence'
            );
            cy.get(topNavSelectors.menuList.userEmail).should('contain.text', 'ai@stackrox.com');
            cy.get(topNavSelectors.menuList.userRoles).should('contain.text', 'Admin');
        });

        it('should show username when name is missed, and all roles', () => {
            mockWithMultiRolesUser();
            cy.visit(dashboardURL);
            cy.get(topNavSelectors.menuButton).click();

            // name is intentionally missed for this mock data, therefore UI should show username
            cy.get(topNavSelectors.menuList.userName).should('contain.text', 'ai');
            cy.get(topNavSelectors.menuList.userEmail).should('contain.text', 'ai@stackrox.com');

            cy.get(topNavSelectors.menuList.userRoles).should('contain.text', 'Admin');
            cy.get(topNavSelectors.menuList.userRoles).should('contain.text', 'Analyst');
            cy.get(topNavSelectors.menuList.userRoles).should(
                'contain.text',
                'Continuous Integration'
            );
        });

        it('should navigate to the user page', () => {
            cy.visit(dashboardURL);
            cy.get(topNavSelectors.menuButton).click();
            cy.get(topNavSelectors.menuList.userName).click();
            cy.url().should('include', userPageUrl);
        });
    });

    describe('User Page', () => {
        it('should show user name & email in the header', () => {
            mockWithAdminUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.pageHeader).should('contain.text', 'Artificial Intelligence');
            cy.get(userPageSelectors.pageHeader).should('contain.text', 'ai@stackrox.com');
        });

        it('should show all the user roles', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(`${userPageSelectors.rolesPanel.table.cells}:contains("Admin")`);
            cy.get(`${userPageSelectors.rolesPanel.table.cells}:contains("Analyst")`);
            cy.get(
                `${userPageSelectors.rolesPanel.table.cells}:contains("Continuous Integration")`
            );
        });

        it('should show correct permissions for the role', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(`${userPageSelectors.rolesPanel.table.cells}:contains("Analyst")`).click();

            // check that read is allowed and write is forbidden
            cy.get(
                userPageSelectors.permissionsPanel.permissionsMatrix.allowedIcon('User', 'read')
            );
            cy.get(
                userPageSelectors.permissionsPanel.permissionsMatrix.forbiddenIcon('User', 'write')
            );
        });

        it('should properly highlight user permissions by role button', () => {
            cy.visit(userPageUrl);
            const highlightedCssClass = 'bg-tertiary-200';
            function checkUserPermissionsHighlightedAndShown() {
                cy.get(userPageSelectors.userPermissionsButton).should(
                    'have.class',
                    highlightedCssClass
                );
                cy.get(userPageSelectors.permissionsPanel.header).should(
                    'contain.text',
                    'User Permissions'
                );
            }

            // it should happen when first time landing on a page
            checkUserPermissionsHighlightedAndShown();

            cy.get(userPageSelectors.rolesPanel.table.row.firstRow).click();
            cy.get(userPageSelectors.userPermissionsButton).should(
                'not.have.class',
                highlightedCssClass
            );

            cy.get(userPageSelectors.userPermissionsButton).click();
            checkUserPermissionsHighlightedAndShown();
        });

        it('should display aggregated permissions for basic auth user', () => {
            mockWithBasicAuthUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.permissionsPanel.roleNameHeader).should(
                'contain.text',
                'admin'
            );
            cy.get(userPageSelectors.permissionsPanel.roleNameHeader).should(
                'contain.text',
                'Basic'
            );
        });

        it('should show correct aggregated permissions', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.permissionsPanel.roleNameHeader).should('contain.text', 'ai');
            cy.get(userPageSelectors.permissionsPanel.roleNameHeader).should(
                'contain.text',
                'My OIDC Provider'
            );

            cy.get(
                userPageSelectors.permissionsPanel.permissionsMatrix.permissionColumn(
                    'User',
                    'read'
                )
            ).should('contain.text', 'Admin, Analyst');
            cy.get(
                userPageSelectors.permissionsPanel.permissionsMatrix.permissionColumn(
                    'User',
                    'write'
                )
            ).should('contain.text', 'Admin');
        });
    });
});
