import scopeSelectors from '../helpers/scopeSelectors';
import selectSelectors from '../selectors/select';

export const url = '/main/access';

const authProviderPanel = '[data-testid="auth-provider-panel"]';

export const selectors = {
    roles: '.rt-tr > .rt-td',
    permissionsPanel: 'div[data-testid=panel]:nth(1)',
    permissionsPanelHeader: 'div[data-testid=panel]:nth(1) div[data-testid=panel-header]',
    permissionsMatrix: scopeSelectors('[data-testid="permissions-matrix"]', {
        rowByPermission: (permission) => `tr:contains("${permission}")`,
    }),
    editButton: 'button:contains("Edit")',
    saveButton: 'button:contains("Save")',
    cancelButton: 'button:contains("Cancel")',
    addNewRoleButton: 'button:contains("Add New Role")',
    input: {
        roleName: 'div[data-testid="role-name"] input',
        issuer: '[data-testid=labeled-key-value-pair]:contains("Issuer") input',
    },
    tabs: {
        authProviders: '[data-testid="tab"]:contains("Auth Provider Rules")',
        roles: '[data-testid="tab"]:contains("Roles and Permissions")',
    },
    message: '.pf-c-alert',
    modal: {
        deleteButton: 'div.ReactModalPortal button:contains("Delete")',
    },
    authProviders: {
        leftSidePanel: {
            selectedRow: 'div[data-testid=panel] div.row-active',
            selectedRowDeleteButton: 'div[data-testid=panel] div.row-active button',
            secondRow: 'div[data-testid=panel] div.rt-tr:nth(2)',
            secondRowDeleteButton: 'div[data-testid=panel] div.rt-tr:nth(2) button',
            thirdRow: 'div[data-testid=panel] div.rt-tr:nth(3)',
        },
        addProviderSelect: selectSelectors.singleSelect,
        newAuth0Option: 'Auth0',
        newIAPOption: 'iap',
        newOidcOption: 'OpenID Connect',
        authProviderPanel: '[data-testid="auth-provider-panel"]',
        authProviderPanelHeader: '[data-testid="auth-provider-panel-header"]',
        clientSecretLabel: 'p:contains("Client Secret")',
        doNotUseClientSecretCheckbox: 'input[name="config.do_not_use_client_secret"]',
        clientSecretInput: 'input[name="config.client_secret"]',
        fragmentCallbackRadio: '#fragment-radio',
        httpPostCallbackRadio: '#post-radio',
    },
    authProviderDetails: scopeSelectors(authProviderPanel, {
        clientSecret: '[data-testid=labeled-key-value-pair]:contains("Client Secret")',
        issuer: '[data-testid=labeled-key-value-pair]:contains("Issuer")',
    }),
};
