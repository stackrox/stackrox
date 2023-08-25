import scopeSelectors from '../helpers/scopeSelectors';

export const selectors = {
    orchestratorComponentsToggle: scopeSelectors('[data-testid="orchestrator-components-toggle"]', {
        hideButton: 'button:contains("Hide")',
        showButton: 'button:contains("Show")',
    }),
    menuButton: '[aria-label="User menu"]',

    menuList: scopeSelectors('[aria-label="User menu"] + ul', {
        logoutButton: 'button:contains("Log out")',
        userEmail: '[data-testid="menu-user-email"]',
        userName: '[data-testid="menu-user-name"]',
        userRoles: '[data-testid="menu-user-roles"]',
    }),
};
