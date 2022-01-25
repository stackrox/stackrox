export const url = '/main/policies-pf';

export const selectors = {
    table: {
        createButton: 'button:contains("Create policy")',
        importButton: 'button:contains("Import policy")',
        policyLink: 'td[data-label="Policy"] a',
        actionsToggleButton: 'td.pf-c-table__action button.pf-c-dropdown__toggle',
        actionsItemButton: 'td.pf-c-table__action ul li[role="menuitem"] button',
    },
    page: {
        actionsToggleButton: 'button.pf-c-dropdown__toggle:contains("Actions")',
        actionsItemButton:
            'button.pf-c-dropdown__toggle:contains("Actions") + ul li[role="menuitem"] button',
    },
    toast: {
        title: 'ul.pf-c-alert-group .pf-c-alert__title',
        description: 'ul.pf-c-alert-group .pf-c-alert__description',
    },
    wizard: {},
    importUploadModal: {
        titleText: '.pf-c-modal-box__title-text:contains("Import policy JSON")',
        fileInput: '.pf-c-file-upload input[type="file"]',
        policyNames: '[data-testid="policies-to-import"] div',
        beginButton: '.pf-c-modal-box__footer button:contains("Begin import")',
        resumeButton: '.pf-c-modal-box__footer button:contains("Resume import")',
        cancelButton: '.pf-c-modal-box__footer button:contains("Cancel")',
        // Form for duplicate policy name
        duplicateAlertTitle:
            '.pf-c-modal-box__body .pf-c-alert__title:contains("Policies already exist")',
        duplicateIdSubstring: '.pf-c-alert__description li:contains("has the same ID")',
        duplicateNameSubstring: '.pf-c-alert__description li:contains("has the same name")',
        keepBothRadioLabel: 'label[for="keep-both-radio"]:contains("Keep both policies")',
        renameRadioLabel: 'label[for="policy-rename-radio"]:contains("Rename incoming policy")',
        renameInput: 'input#policy-rename',
        overwriteRadioLabel:
            'label[for="policy-overwrite-radio-1"]:contains("Overwrite existing policy")',
    },
    importSuccessModal: {
        titleText: '.pf-c-modal-box__title-text:contains("Import policy JSON")',
        policyNames: '[data-testid="policies-imported"] div',
    },
};
