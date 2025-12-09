import pageSelectors from '../selectors/page';
import scopeSelectors from '../helpers/scopeSelectors';

const permissionColumn = (permission, testid) => {
    return `tr:contains("${permission}") td[data-testid="${testid}"]`;
};

export const selectors = {
    pageHeader: pageSelectors.pageHeader,

    // DescriptionList
    userName: 'dl div:contains("User name") dd',
    userEmail: 'dl div:contains("User email") dd',
    authProviderName: 'dl div:contains("Auth provider") dd',

    // Select
    userRoleSelector: '#user-role-selector',

    // Select Options
    userPermissionsForRoles:
        '[role="listbox"] [role="option"]:contains("User permissions for all roles")',
    userRoleNames: '[role="listbox"] [role="option"]',

    // Table
    permissionsTable: scopeSelectors('table', {
        permissionColumn,
        /** allowed icon selector by permission name and access: {read | write} */
        allowedIcon: (permission, testid) =>
            `${permissionColumn(permission, testid)} [aria-label="permitted"]`,
        /** forbidden icon selector by permission name and access: {read | write} */
        forbiddenIcon: (permission, testid) =>
            `${permissionColumn(permission, testid)} [aria-label="forbidden"]`,
    }),
};
