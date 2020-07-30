import pageSelectors from '../selectors/page';
import tableSelectors from '../selectors/table';
import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/user';

const permissionColumn = (permission, access) => {
    if (access !== 'read' && access !== 'write') throw new Error(`Invalid access param: ${access}`); // miss TypeScript...
    const columnNumber = access === 'read' ? 1 : 2;
    return `tr:contains("${permission}") td:eq(${columnNumber})`;
};

export const selectors = {
    pageHeader: pageSelectors.pageHeader,

    userPermissionsButton: 'button:contains("StackRox User Permissions")',

    rolesPanel: scopeSelectors('[data-testid="panel"]:contains("StackRox User Roles")', {
        table: tableSelectors,
    }),

    permissionsPanel: scopeSelectors('[data-testid="panel"]:contains("Permissions")', {
        header: '[data-testid="panel-header"]',
        roleNameHeader: '[data-testid="role-name-header"]',
        permissionsMatrix: scopeSelectors('[data-testid="permissions-matrix"]', {
            permissionColumn,
            /** allowed icon selector by permission name and access: {read | write} */
            allowedIcon: (permission, access) =>
                `${permissionColumn(permission, access)} .text-success-600`,
            /** forbidden icon selector by permission name and access: {read | write} */
            forbiddenIcon: (permission, access) =>
                `${permissionColumn(permission, access)} .text-alert-600`,
        }),
    }),
};
