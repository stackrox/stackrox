import withAuth from '../helpers/basicAuth';
import { selectors as userPageSelectors, url as userPageUrl } from '../constants/UserPage';
import { url as dashboardURL } from '../constants/DashboardPage';
import { selectors as topNavSelectors } from '../constants/TopNavigation';
import * as api from '../constants/apiEndpoints';

describe('User Info', () => {
    withAuth();

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
        it('should show user name and email', () => {
            mockWithAdminUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.userName).should('contain.text', 'Artificial Intelligence');
            cy.get(userPageSelectors.userEmail).should('contain.text', 'ai@stackrox.com');
        });

        it('should show all the user roles', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(`${userPageSelectors.userRoleNames}:contains("Admin")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Continuous Integration")`);
        });

        it('should show correct permissions for the role', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`).click();

            // check that read is allowed and write is forbidden
            cy.get(userPageSelectors.permissionsTable.allowedIcon('User', 'read'));
            cy.get(userPageSelectors.permissionsTable.forbiddenIcon('User', 'write'));
        });

        it('should properly highlight current nav item', () => {
            cy.visit(userPageUrl);

            const { userPermissionsForRoles, userRoleNames } = userPageSelectors;
            const userRoleAdmin = `${userRoleNames}:contains("Admin")`;
            const currentClass = 'pf-m-current';

            // When landing on Users page:
            cy.get(userPermissionsForRoles).should('have.class', currentClass);
            cy.get(userRoleAdmin).should('not.have.class', currentClass);

            // After clicking Admin user role:
            cy.get(userRoleAdmin).click();
            cy.get(userPermissionsForRoles).should('not.have.class', currentClass);
            cy.get(userRoleAdmin).should('have.class', currentClass);

            // After clicking User permissions for roles:
            cy.get(userPermissionsForRoles).click();
            cy.get(userPermissionsForRoles).should('have.class', currentClass);
            cy.get(userRoleAdmin).should('not.have.class', currentClass);
        });

        it('should display aggregated permissions for basic auth user', () => {
            mockWithBasicAuthUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.userName).should('contain.text', 'admin');
            cy.get(userPageSelectors.authProviderName).should('contain.text', 'Basic');

            cy.get(userPageSelectors.permissionsTable.permissionColumn('User', 'read'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
            cy.get(userPageSelectors.permissionsTable.permissionColumn('User', 'write'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
        });

        it('should show correct aggregated permissions for multi roles user', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(userPageSelectors.userName).should('contain.text', 'ai');
            cy.get(userPageSelectors.authProviderName).should('contain.text', 'My OIDC Provider');

            cy.get(userPageSelectors.permissionsTable.permissionColumn('User', 'read'))
                .should('contain.text', 'Admin')
                .should('contain.text', 'Analyst');
            cy.get(userPageSelectors.permissionsTable.permissionColumn('User', 'write'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
        });
    });
});
