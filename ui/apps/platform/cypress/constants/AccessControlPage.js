import scopeSelectors from '../helpers/scopeSelectors';

export const accessControlUrl = '/main/access-control';
export const authProvidersUrl = '/main/access-control/auth-providers';
export const rolesUrl = '/main/access-control/roles';
export const permissionSetsUrl = '/main/access-control/permission-sets';
export const accessScopesUrl = '/main/access-control/access-scopes';

function getFormGroupControlForLabel(label) {
    return `.pf-c-form__group-label:contains("${label}") + .pf-c-form__group-control`;
}

export const selectors = scopeSelectors('#access-control', {
    breadcrumbItem: '.pf-c-breadcrumb__item',
    breadcrumbLink: 'a.pf-c-breadcrumb__link',
    h1: 'h1',
    h2: 'h2',
    navLink: 'nav a',
    navLinkCurrent: 'nav a.pf-m-current',
    alertTitle: '.pf-c-alert__title',

    list: {
        addButton: 'button:contains("Add")',
        th: 'th',
        tdNameLink: 'td[data-label="Name"] button',
        tdDescription: 'td[data-label="Description"]',

        authProviders: {
            addDropdownItem: 'button:contains("Add auth provider") + ul button',
            tdType: 'td[data-label="Type"]',
            tdMinimumAccessRole: 'td[data-label="Minimum access role',
            tdRules: 'td[data-label="Rules"]',
        },

        roles: {
            tdPermissionSetLink: 'td[data-label="Permission set"] button a',
            tdAccessScopeLink: 'td[data-label="Access scope"] button a',
            tdAccessScope: 'td[data-label="Access scope"]', // No access scope
        },

        permissionSets: {
            tdRolesLink: 'td[data-label="Roles"] button a',
            tdRoles: 'td[data-label="Roles"]', // No roles
        },

        accessScopes: {
            tdRolesLink: 'td[data-label="Roles"] ',
            tdRoles: 'td[data-label="Roles"]', // No roles
        },
    },

    form: {
        notEditableLabel: '.pf-c-label:contains("Not editable")',
        editButton: 'button:contains("Edit")',
        saveButton: 'button:contains("Save")',
        cancelButton: 'button:contains("Cancel")',

        inputName: `${getFormGroupControlForLabel('Name')} input`,
        inputDescription: `${getFormGroupControlForLabel('Description')} input`,

        authProvider: scopeSelectors('form', {
            selectAuthProviderType: `${getFormGroupControlForLabel(
                'Auth provider type'
            )} .pf-c-select button`,

            auth0: {
                inputAuth0Tenant: `${getFormGroupControlForLabel('Auth0 tenant')} input`,
                inputClientID: `${getFormGroupControlForLabel('Client ID')} input`,
            },
            oidc: {
                selectCallbackMode: `${getFormGroupControlForLabel(
                    'Callback mode'
                )} .pf-c-select button`,
                selectCallbackModeItem: `${getFormGroupControlForLabel(
                    'Callback mode'
                )} .pf-c-select button + ul button`,
                inputIssuer: `${getFormGroupControlForLabel('Issuer')} input`,
                inputClientID: `${getFormGroupControlForLabel('Client ID')} input`,
                inputClientSecret: `${getFormGroupControlForLabel('Client Secret')} input`, // TODO sentence case?
                checkboxDoNotUseClientSecret:
                    '.pf-c-check:contains("Do not use Client Secret") input[type="checkbox"]',
            },
            saml: {
                inputServiceProviderIssuer: `${getFormGroupControlForLabel(
                    'Service Provider issuer'
                )} input`, // TODO sentence case?
                selectConfiguration: `${getFormGroupControlForLabel(
                    'Configuration'
                )} .pf-c-select button`,
                inputMetadataURL: `${getFormGroupControlForLabel('IdP Metadata URL')} input`, // TODO sentence case?
            },
            userpki: {
                textareaCertificates: `${getFormGroupControlForLabel(
                    'IdP Certificate(s) (PEM)'
                )} textarea`, // TODO sentence case?
            },
            iap: {
                inputAudience: `${getFormGroupControlForLabel('Audience')} input`,
            },
        }),

        role: scopeSelectors('#role-form', {
            getRadioPermissionSetForName: (name) =>
                `.pf-c-form__group-label:contains("Permission set") + .pf-c-form__group-control tr:contains("${name}") input[type="radio"]`,
            getRadioAccessScopeForName: (name) =>
                `.pf-c-form__group-label:contains("Access scope") + .pf-c-form__group-control tr:contains("${name}") input[type="radio"]`,
        }),

        permissionSet: scopeSelectors('#permission-set-form', {
            resourceCount: 'th:contains("Resource") .pf-c-badge',
            readCount: 'th:contains("Read") .pf-c-badge',
            writeCount: 'th:contains("Write") .pf-c-badge',
            tdResource: 'td[data-label="Resource"]',

            // Zero-based index for Image instead of ImageComponent, ImageIntegration, WatchedImage.
            getReadAccessIconForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:contains("${resource}"):eq(${index}) ~ td[data-label="Read"] svg`,
            getWriteAccessIconForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:contains("${resource}"):eq(${index}) ~ td[data-label="Write"] svg`,
            getAccessLevelSelectForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:contains("${resource}"):eq(${index}) ~ td[data-label="Access level"] .pf-c-select__toggle`,
        }),

        accessScope: scopeSelectors('#access-scope-form', {}),
    },
});
