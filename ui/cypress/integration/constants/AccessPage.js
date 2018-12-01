export const url = '/main/access';

export const selectors = {
    roles: '.rt-tr > .rt-td',
    permissionsPanel: 'div[data-test-id=panel]:nth(1)',
    permissionsPanelHeader: 'div[data-test-id=panel]:nth(1) div[data-test-id=panel-header]',
    editButton: 'button:contains("Edit")',
    saveButton: 'button:contains("Save")',
    addNewRoleButton: 'button:contains("Add New Role")',
    input: {
        roleName: 'input[type=text]'
    }
};
