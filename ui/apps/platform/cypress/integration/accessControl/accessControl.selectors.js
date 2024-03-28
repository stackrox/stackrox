import scopeSelectors from '../../helpers/scopeSelectors';

function getFormGroupControlForLabel(label) {
    return `.pf-v5-c-form__group-label:contains("${label}") + .pf-v5-c-form__group-control`;
}

export const selectors = scopeSelectors('main', {
    alertTitle: '.pf-v5-c-alert__title',

    list: {
        authProviders: {
            dataRows: 'tbody tr',
            createDropdownItem: 'button:contains("Create auth provider") + ul button',
            tdType: 'td[data-label="Type"]',
            tdMinimumAccessRole: 'td[data-label="Minimum access role',
            tdRules: 'td[data-label="Rules"]',
        },

        roles: {
            tdPermissionSetLink: 'td[data-label="Permission set"] a',
            tdAccessScopeLink: 'td[data-label="Access scope"] a',
            tdAccessScope: 'td[data-label="Access scope"]', // No access scope
        },

        permissionSets: {
            tdRolesLink: 'td[data-label="Roles"] a',
            tdRoles: 'td[data-label="Roles"]', // No roles
        },

        accessScopes: {
            tdRolesLink: 'td[data-label="Roles"] a',
            tdRoles: 'td[data-label="Roles"]', // No roles
        },
    },

    form: {
        notEditableLabel: '.pf-v5-c-label:contains("Not editable")',
        editButton: 'button:contains("Edit")',
        saveButton: 'button:contains("Save")',
        cancelButton: 'button:contains("Cancel")',

        inputName: `${getFormGroupControlForLabel('Name')} input`,
        inputDescription: `${getFormGroupControlForLabel('Description')} input`,

        authProvider: scopeSelectors('form', {
            selectAuthProviderType: `${getFormGroupControlForLabel(
                'Auth provider type'
            )} .pf-v5-c-select button`,

            auth0: {
                inputAuth0Tenant: `${getFormGroupControlForLabel('Auth0 tenant')} input`,
                inputClientID: `${getFormGroupControlForLabel('Client ID')} input`,
            },
            oidc: {
                selectCallbackMode: `${getFormGroupControlForLabel(
                    'Callback mode'
                )} .pf-v5-c-select button`,
                selectCallbackModeItem: `${getFormGroupControlForLabel(
                    'Callback mode'
                )} .pf-v5-c-select button + ul button`,
                inputIssuer: `${getFormGroupControlForLabel('Issuer')} input`,
                inputClientID: `${getFormGroupControlForLabel('Client ID')} input`,
                inputClientSecret: `${getFormGroupControlForLabel('Client Secret')} input`, // TODO sentence case?
                checkboxDoNotUseClientSecret:
                    '.pf-v5-c-check:contains("Do not use Client Secret") input[type="checkbox"]',
            },
            saml: {
                inputServiceProviderIssuer: `${getFormGroupControlForLabel(
                    'Service Provider issuer'
                )} input`, // TODO sentence case?
                selectConfiguration: `${getFormGroupControlForLabel(
                    'Configuration'
                )} .pf-v5-c-select button`,
                inputMetadataURL: `${getFormGroupControlForLabel('IdP Metadata URL')} input`, // TODO sentence case?
            },
            userpki: {
                textareaCertificates: `${getFormGroupControlForLabel(
                    'CA certificate(s) (PEM)'
                )} textarea`,
            },
            iap: {
                inputAudience: `${getFormGroupControlForLabel('Audience')} input`,
            },
        }),

        minimumAccessRole: scopeSelectors('form', {
            selectMinimumAccessRole: `${getFormGroupControlForLabel(
                'Minimum access role'
            )} .pf-v5-c-select button`,
            selectMinimumAccessRoleItem: `${getFormGroupControlForLabel(
                'Minimum access role'
            )} .pf-v5-c-select button + ul button`,
        }),

        role: scopeSelectors('#role-form', {
            getRadioPermissionSetForName: (name) =>
                `.pf-v5-c-form__group-label:contains("Permission set") + .pf-v5-c-form__group-control tr:contains("${name}") input[type="radio"]`,
            getRadioAccessScopeForName: (name) =>
                `.pf-v5-c-form__group-label:contains("Access scope") + .pf-v5-c-form__group-control tr:contains("${name}") input[type="radio"]`,
        }),

        permissionSet: scopeSelectors('#permission-set-form', {
            resourceCount: 'th:contains("Resource") .pf-v5-c-badge',
            readCount: 'th:contains("Read") .pf-v5-c-badge',
            writeCount: 'th:contains("Write") .pf-v5-c-badge',
            tdResource: 'td[data-label="Resource"] p:first-child',

            // Zero-based index for Image instead of ImageComponent, ImageIntegration, WatchedImage.
            getReadAccessIconForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:has('p:first-child:contains("${resource}")'):eq(${index}) ~ td[data-label="Read"] svg`,
            getWriteAccessIconForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:has('p:first-child:contains("${resource}")'):eq(${index}) ~ td[data-label="Write"] svg`,
            getAccessLevelSelectForResource: (resource, index = 0) =>
                `td[data-label="Resource"]:has('p:first-child:contains("${resource}")'):eq(${index}) ~ td[data-label="Access level"] .pf-v5-c-select__toggle`,
        }),

        accessScope: scopeSelectors('#access-scope-form', {}),
    },
});

export const accessModalSelectors = {
    title: '.pf-v5-c-modal-box__title-text',
    body: '.pf-v5-c-modal-box__body',
    button: '.pf-v5-c-modal-box__footer button',
    cancel: '.pf-v5-c-modal-box__footer button:contains("Cancel")',
    delete: '.pf-v5-c-modal-box__footer button:contains("Delete")',
};
