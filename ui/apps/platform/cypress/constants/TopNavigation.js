import scopeSelectors from '../helpers/scopeSelectors';

// eslint-disable-next-line import/prefer-default-export
export const selectors = {
    menuButton: '[data-testid="menu-button"]',

    menuList: scopeSelectors('[data-testid="menu-list"]', {
        logoutButton: 'button:contains("Logout")',
        userEmail: '[data-testid="menu-user-email"]',
        userName: '[data-testid="menu-user-name"]',
        userRoles: '[data-testid="menu-user-roles"]',
    }),
};
