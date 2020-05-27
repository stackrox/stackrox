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

            cy.get(`${userPageSelectors.rolesSidePanel.table.cells}:contains("Admin")`);
            cy.get(`${userPageSelectors.rolesSidePanel.table.cells}:contains("Analyst")`);
            cy.get(
                `${userPageSelectors.rolesSidePanel.table.cells}:contains("Continuous Integration")`
            );
        });

        it('should show correct permissions for the role', () => {
            mockWithMultiRolesUser();
            cy.visit(userPageUrl);

            cy.get(`${userPageSelectors.rolesSidePanel.table.cells}:contains("Analyst")`).click();

            // check that read is allowed and write is forbidden
            cy.get(userPageSelectors.permissionsMatrix.allowedIcon('User', 'read'));
            cy.get(userPageSelectors.permissionsMatrix.forbiddenIcon('User', 'write'));
        });
    });
});
