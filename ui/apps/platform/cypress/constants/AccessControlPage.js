import scopeSelectors from '../helpers/scopeSelectors';

export const accessControlUrl = '/main/access-control';
export const authProvidersUrl = '/main/access-control/auth-providers';
export const rolesUrl = '/main/access-control/roles';
export const permissionSetsUrl = '/main/access-control/permission-sets';
export const accessScopesUrl = '/main/access-control/access-scopes';

export const selectors = scopeSelectors('#access-control', {
    h1: 'h1',
    h2: 'h2',
    navLink: 'nav a',
    navLinkCurrent: 'nav a.pf-m-current',
    alertTitle: '.pf-c-alert__title',

    list: {
        addButton: 'button:contains("Add")',
        th: 'th',
        tdLinkName: 'td[data-label="Name"] button',
        tdDescription: 'td[data-label="Description"]',

        authProviders: {
            tdType: 'td[data-label="Type"]',
            tdMinimumAccessRole: 'td[data-label="Minimum access role',
            tdRules: 'td[data-label="Rules"]',
        },

        roles: {
            tdPermissionSet: 'td[data-label="Permission set"]',
            tdAccessScope: 'td[data-label="Access scope"]',
        },

        permissionSets: {
            tdRoles: 'td[data-label="Roles"]',
        },

        accessScopes: {
            tdRoles: 'td[data-label="Roles"]',
        },
    },

    form: {
        notEditableLabel: '.pf-c-label:contains("Not editable")',
        editButton: 'button:contains("Edit")',
        saveButton: 'button:contains("Save")',
        cancelButton: 'button:contains("Cancel")',

        inputName: '.pf-c-form__group-label:contains("Name") + .pf-c-form__group-control input',
        inputDescription:
            '.pf-c-form__group-label:contains("Description") + .pf-c-form__group-control input',

        authProvider: scopeSelectors('form', {}),

        role: scopeSelectors('#role-form', {
            getRadioPermissionSetForName: (name) =>
                `.pf-c-form__group-label:contains("Permission set") + .pf-c-form__group-control tr:contains("${name}") input[type="radio"]`,
            getRadioAccessScopeForName: (name) =>
                `.pf-c-form__group-label:contains("Access scope") + .pf-c-form__group-control tr:contains("${name}") input[type="radio"]`,
        }),

        permissionSet: scopeSelectors('#permission-set-form', {
            readCount: 'th:contains("Read") .pf-c-badge',
            writeCount: 'th:contains("Write") .pf-c-badge',
            getAccessLevelForResourceName: (resourceName) =>
                `.pf-c-form__group-label:contains("Permissions") + .pf-c-form__group-control tr:contains("${resourceName}") .pf-c-select__toggle-text`,
        }),

        accessScope: scopeSelectors('#access-scope-form', {}),
    },
});
