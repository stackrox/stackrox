import scopeSelectors from '../helpers/scopeSelectors';
import pf6 from '../selectors/pf6';

export const selectors = {
    menuButton: `${pf6.menuToggle}[aria-label="User menu"]`,
    menuList: scopeSelectors(pf6.dropdownItem, {
        userEmail: '[data-testid="menu-user-email"]',
        userName: '[data-testid="menu-user-name"]',
        userRoles: '[data-testid="menu-user-roles"]',
        myProfileButton: 'button:contains("My profile")',
        inviteUsersButton: 'button:contains("Invite users")',
        logoutButton: 'button:contains("Log out")',
    }),
};
