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

    // Nav
    userPermissionsForRoles: 'nav[aria-label="Roles"] li:contains("User permissions for roles") a',
    userRoleNames: 'nav[aria-label="Roles"] li:contains("User roles") li a',

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
