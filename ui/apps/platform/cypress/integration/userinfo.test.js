import { selectors as userPageSelectors } from '../constants/UserPage';
import { selectors as topNavSelectors } from '../constants/TopNavigation';
import withAuth from '../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../helpers/title';
import { checkInviteUsersModal } from '../helpers/inviteUsers';
import { closeModalByButton } from '../helpers/modal';
import {
    visitUserProfile,
    visitUserProfileFromTopNav,
    visitUserProfileWithStaticResponseForAuthStatus,
} from '../helpers/user';
import {
    authProvidersAlias,
    rolesAlias,
    visitAccessControlEntities,
} from './accessControl/accessControl.helpers';

const staticResponseForAdminRoleWithoutProvider = {
    fixture: 'auth/adminUserStatus',
};

const staticResponseForMultiRolesWithOidcProvider = {
    fixture: 'auth/multiRolesUserStatus',
};

const staticResponseForAdminRoleWithBasicProvider = {
    fixture: 'auth/basicAuthAdminStatus',
};

describe('User Profile', () => {
    withAuth();

    describe('in top navigation', () => {
        it('should show initials in the user avatar', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForAdminRoleWithoutProvider
            );

            cy.get(topNavSelectors.menuButton).should('contain.text', 'AI');
        });

        it('should show name, email and a single role', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForAdminRoleWithoutProvider
            );

            cy.get(topNavSelectors.menuButton).click();

            cy.get(topNavSelectors.menuList.userName).should(
                'contain.text',
                'Artificial Intelligence'
            );
            cy.get(topNavSelectors.menuList.userEmail).should('contain.text', 'ai@stackrox.com');
            cy.get(topNavSelectors.menuList.userRoles).should('contain.text', 'Admin');
        });

        it('should show username when name is missed, and all roles', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForMultiRolesWithOidcProvider
            );

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

        it('should have a trigger for opening the Invite users modal', () => {
            const staticResponseMap = {
                [authProvidersAlias]: {
                    fixture: 'auth/authProviders-id1-id3.json',
                },
                [rolesAlias]: {
                    fixture: 'auth/roles.json',
                },
            };
            visitAccessControlEntities('roles', staticResponseMap); // page doens't matter because user menu is on every page

            // open menu and click Invite useres menu item
            cy.get(topNavSelectors.menuButton).click();
            cy.get('.pf-c-dropdown__menu-item:contains("Invite users")').click();

            checkInviteUsersModal();

            // test closing the modal
            closeModalByButton();
        });

        it('should navigate to the user page', () => {
            visitUserProfileFromTopNav();
        });
    });

    describe('page', () => {
        it('should have title', () => {
            visitUserProfile();

            cy.title().should('match', getRegExpForTitleWithBranding('User Profile'));
        });

        it('should show user name and email', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForAdminRoleWithoutProvider
            );

            cy.get(userPageSelectors.userName).should('contain.text', 'Artificial Intelligence');
            cy.get(userPageSelectors.userEmail).should('contain.text', 'ai@stackrox.com');
        });

        it('should show all the user roles', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForMultiRolesWithOidcProvider
            );

            cy.get(`${userPageSelectors.userRoleNames}:contains("Admin")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`);
            cy.get(`${userPageSelectors.userRoleNames}:contains("Continuous Integration")`);
        });

        it('should show correct permissions for the role', () => {
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForMultiRolesWithOidcProvider
            );

            cy.get(`${userPageSelectors.userRoleNames}:contains("Analyst")`).click();

            // check that read is allowed and write is forbidden
            cy.get(userPageSelectors.permissionsTable.allowedIcon('Access', 'read'));
            cy.get(userPageSelectors.permissionsTable.forbiddenIcon('Access', 'write'));
        });

        it('should properly highlight current nav item', () => {
            visitUserProfile();

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
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForAdminRoleWithBasicProvider
            );

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
            visitUserProfileWithStaticResponseForAuthStatus(
                staticResponseForMultiRolesWithOidcProvider
            );

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
