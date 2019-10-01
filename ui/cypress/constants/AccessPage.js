export const url = '/main/access';

export const selectors = {
    roles: '.rt-tr > .rt-td',
    permissionsPanel: 'div[data-test-id=panel]:nth(1)',
    permissionsPanelHeader: 'div[data-test-id=panel]:nth(1) div[data-test-id=panel-header]',
    editButton: 'button:contains("Edit")',
    saveButton: 'button:contains("Save")',
    addNewRoleButton: 'button:contains("Add New Role")',
    input: {
        roleName: 'div[data-test-id="role-name"] input'
    },
    tabs: {
        authProviders: '[data-test-id="tab"]:contains("Auth Provider Rules")',
        roles: '[data-test-id="tab"]:contains("Roles and Permissions")'
    },
    authProviders: {
        addProvider: 'select:contains("Add an Auth Provider")',
        newAuth0: 'auth0',
        newAuthProviderPanel: '[data-test-id="auth-provider-panel"]'
    }
};
