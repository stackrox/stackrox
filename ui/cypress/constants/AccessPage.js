export const url = '/main/access';

export const selectors = {
    roles: '.rt-tr > .rt-td',
    permissionsPanel: 'div[data-testid=panel]:nth(1)',
    permissionsPanelHeader: 'div[data-testid=panel]:nth(1) div[data-testid=panel-header]',
    editButton: 'button:contains("Edit")',
    saveButton: 'button:contains("Save")',
    cancelButton: 'button:contains("Cancel")',
    addNewRoleButton: 'button:contains("Add New Role")',
    input: {
        roleName: 'div[data-testid="role-name"] input'
    },
    tabs: {
        authProviders: '[data-testid="tab"]:contains("Auth Provider Rules")',
        roles: '[data-testid="tab"]:contains("Roles and Permissions")'
    },
    modal: {
        deleteButton: 'div.ReactModalPortal button:contains("Delete")'
    },
    authProviders: {
        leftSidePanel: {
            selectedRow: 'div[data-testid=panel] div.row-active',
            selectedRowDeleteButton: 'div[data-testid=panel] div.row-active button',
            secondRow: 'div[data-testid=panel] div.rt-tr:nth(2)',
            secondRowDeleteButton: 'div[data-testid=panel] div.rt-tr:nth(2) button',
            thirdRow: 'div[data-testid=panel] div.rt-tr:nth(3)'
        },
        addProviderSelect: 'select:contains("Add an Auth Provider")',
        newAuth0Option: 'auth0',
        newOidcOption: 'OpenID Connect',
        authProviderPanel: '[data-testid="auth-provider-panel"]',
        authProviderPanelHeader: '[data-testid="auth-provider-panel-header"]',
        clientSecretLabel: 'p:contains("Client Secret")',
        doNotUseClientSecretCheckbox: 'input[name="config.do_not_use_client_secret"]',
        clientSecretInput: 'input[name="config.client_secret"]',
        fragmentCallbackRadio: '#fragment-radio',
        httpPostCallbackRadio: '#post-radio'
    }
};
