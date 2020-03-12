export const url = '/main/access';

export const selectors = {
    roles: '.rt-tr > .rt-td',
    permissionsPanel: 'div[data-test-id=panel]:nth(1)',
    permissionsPanelHeader: 'div[data-test-id=panel]:nth(1) div[data-test-id=panel-header]',
    editButton: 'button:contains("Edit")',
    saveButton: 'button:contains("Save")',
    cancelButton: 'button:contains("Cancel")',
    addNewRoleButton: 'button:contains("Add New Role")',
    input: {
        roleName: 'div[data-test-id="role-name"] input'
    },
    tabs: {
        authProviders: '[data-test-id="tab"]:contains("Auth Provider Rules")',
        roles: '[data-test-id="tab"]:contains("Roles and Permissions")'
    },
    modal: {
        deleteButton: 'div.ReactModalPortal button:contains("Delete")'
    },
    authProviders: {
        leftSidePanel: {
            selectedRow: 'div[data-test-id=panel] div.row-active',
            selectedRowDeleteButton: 'div[data-test-id=panel] div.row-active button',
            secondRow: 'div[data-test-id=panel] div.rt-tr:nth(2)',
            secondRowDeleteButton: 'div[data-test-id=panel] div.rt-tr:nth(2) button',
            thirdRow: 'div[data-test-id=panel] div.rt-tr:nth(3)'
        },
        addProviderSelect: 'select:contains("Add an Auth Provider")',
        newAuth0Option: 'auth0',
        newOidcOption: 'OpenID Connect',
        authProviderPanel: '[data-test-id="auth-provider-panel"]',
        authProviderPanelHeader: '[data-test-id="auth-provider-panel-header"]',
        clientSecretLabel: 'p:contains("Client Secret")',
        doNotUseClientSecretCheckbox: 'input[name="config.do_not_use_client_secret"]',
        clientSecretInput: 'input[name="config.client_secret"]',
        fragmentCallbackRadio: '#fragment-radio',
        httpPostCallbackRadio: '#post-radio'
    }
};
