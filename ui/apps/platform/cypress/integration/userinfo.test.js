import withAuth from '../helpers/basicAuth';
import { selectors as userPageSelectors, url as userPageUrl } from '../constants/UserPage';
import { url as dashboardURL } from '../constants/DashboardPage';
import { selectors as topNavSelectors } from '../constants/TopNavigation';
import * as api from '../constants/apiEndpoints';

describe('User Info', () => {
    withAuth();

    function interceptWithoutMockUser() {
        cy.intercept('GET', api.auth.authStatus).as('authStatus');
    }

    function interceptWithMockAdminUser() {
        cy.intercept('GET', api.auth.authStatus, {
            fixture: 'auth/adminUserStatus',
        }).as('authStatus');
    }

    function interceptWithMockMultiRolesUser() {
        cy.intercept('GET', api.auth.authStatus, {
            fixture: 'auth/multiRolesUserStatus',
        }).as('authStatus');
    }

    function interceptWithMockBasicUser() {
        cy.intercept('GET', api.auth.authStatus, {
            fixture: 'auth/basicAuthAdminStatus',
        }).as('authStatus');
    }

    describe('User Info in Top Navigation', () => {
        it('should show initials in the user avatar', () => {
            interceptWithMockAdminUser();
            cy.visit(dashboardURL);
            cy.wait('@authStatus');

            cy.get(topNavSelectors.menuButton).should('contain.text', 'AI');
        });

        it('should show name, email and a single role', () => {
            interceptWithMockAdminUser();
            cy.visit(dashboardURL);
            cy.wait('@authStatus');

            cy.get(topNavSelectors.menuButton).click();

            cy.get(topNavSelectors.menuList.userName).should(
                'contain.text',
                'Artificial Intelligence'
            );
            cy.get(topNavSelectors.menuList.userEmail).should('contain.text', 'ai@stackrox.com');
            cy.get(topNavSelectors.menuList.userRoles).should('contain.text', 'Admin');
        });

        it('should show username when name is missed, and all roles', () => {
            interceptWithMockMultiRolesUser();
            cy.visit(dashboardURL);
            cy.wait('@authStatus');

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
            interceptWithoutMockUser();
            cy.visit(dashboardURL);
            cy.wait('@authStatus');

            cy.get(topNavSelectors.menuButton).click();
            cy.get(topNavSelectors.menuList.userName).click();
            cy.wait('@authStatus');

            cy.location('pathname').should('eq', userPageUrl);
        });
    });

    describe('User Page', () => {
        // TODO after we split into 2 test files and factor out helper functions.
        /*
        it('should have title', () => {
            visitUserProfile();
    
            cy.title().should('match', getRegExpForTitleWithBranding('User Profile'));
        });
        */

        it('should show user name and email', () => {
            interceptWithMockAdminUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

            cy.get(userPageSelectors.userName).should('contain.text', 'Artificial Intelligence');
            cy.get(userPageSelectors.userEmail).should('contain.text', 'ai@stackrox.com');
        });

        it('should show all the user roles', () => {
            interceptWithMockMultiRolesUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

            cy.get(`${userPageSelectors.userRoleNames}:contains("Admin")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Continuous Integration")`);
        });

        it('should show correct permissions for the role', () => {
            interceptWithMockMultiRolesUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`).click();

            // check that read is allowed and write is forbidden
            cy.get(userPageSelectors.permissionsTable.allowedIcon('Access', 'read'));
            cy.get(userPageSelectors.permissionsTable.forbiddenIcon('Access', 'write'));
        });

        it('should properly highlight current nav item', () => {
            interceptWithoutMockUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

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
            interceptWithMockBasicUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

            cy.get(userPageSelectors.userName).should('contain.text', 'admin');
            cy.get(userPageSelectors.authProviderName).should('contain.text', 'Basic');

            cy.get(userPageSelectors.permissionsTable.permissionColumn('Access', 'read'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
            cy.get(userPageSelectors.permissionsTable.permissionColumn('Access', 'write'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
        });

        it('should show correct aggregated permissions for multi roles user', () => {
            interceptWithMockMultiRolesUser();
            cy.visit(userPageUrl);
            cy.wait('@authStatus');

            cy.get(userPageSelectors.userName).should('contain.text', 'ai');
            cy.get(userPageSelectors.authProviderName).should('contain.text', 'My OIDC Provider');

            cy.get(userPageSelectors.permissionsTable.permissionColumn('Access', 'read'))
                .should('contain.text', 'Admin')
                .should('contain.text', 'Analyst');
            cy.get(userPageSelectors.permissionsTable.permissionColumn('Access', 'write'))
                .should('contain.text', 'Admin')
                .should('not.contain.text', 'Analyst');
        });
    });
});
