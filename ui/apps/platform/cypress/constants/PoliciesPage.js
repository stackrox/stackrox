import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/policy-management/policies';

export const selectors = {
    table: {
        createButton: 'button:contains("Create policy")',
        importButton: 'button:contains("Import policy")',
        searchInput: '.react-select__input > input',
        bulkActionsDropdownButton: 'button:contains("Bulk actions")',
        bulkActionsDropdownItem: 'button:contains("Bulk actions") + ul[role="menu"] li',
        reassessButton: 'button:contains("Reassess all")',
        policyLink: 'td[data-label="Policy"] a',
        statusCell: 'td[data-label="Status"]',
        originCell: 'td[data-label="Origin"]',
        severityCell: 'td[data-label="Severity"]',
        lifecycleCell: 'td[data-label="Lifecycle"]',
        selectCheckbox: '.pf-c-table__check input[type="checkbox"]',
        actionsToggleButton: 'td.pf-c-table__action button.pf-c-dropdown__toggle',
        actionsItemButton: 'td.pf-c-table__action ul li button[role="menuitem"]',
        firstRow: '[data-testid="policies-table"] tbody tr:nth(0)',
        rows: '[data-testid="policies-table"] tbody tr',
    },
    page: {
        actionsToggleButton: 'button.pf-c-dropdown__toggle:contains("Actions")',
        actionsItemButton:
            'button.pf-c-dropdown__toggle:contains("Actions") + ul li button[role="menuitem"]',
    },
    toast: {
        title: 'ul.pf-c-alert-group .pf-c-alert__title',
        description: 'ul.pf-c-alert-group .pf-c-alert__description',
    },
    wizardBtns: {
        step3: '.pf-c-wizard__nav-link:contains("criteria")',
        step5: '.pf-c-wizard__nav-link:contains("Review policy")',
    },
    step3: {
        defaultPolicyAlert: '[data-testid="default-policy-alert"]',
        policySection: {
            addBtn: '[data-testid="add-section-btn"]',
            deleteBtn: '[data-testid="delete-section-btn"]',
            cards: '.policy-section-card-body',
            name: '[data-testid="policy-section-name"]',
            nameEditBtn: '.policy-section-card-header [data-testid="edit-section-name-btn"]',
            nameSaveBtn: '.policy-section-card-header [data-testid="save-section-name-btn"]',
            nameInput: '.policy-section-card-header input',
            orDivider: 'div.or-divider-container',
            dropTarget: '[data-testid="policy-section-drop-target"]',
        },
        policyCriteria: {
            keyGroup: '[data-testid="policy-criteria-key-group"]',
            key: '.policy-criteria-key',
            groupCards: '[data-testid="policy-criteria-group-card"]',
            groupCardTitle:
                '[data-testid="policy-criteria-group-card"] .pf-c-card__header .pf-c-card__title',
            value: {
                textInput: '[data-testid="policy-criteria-value-text-input"]',
                numberInput: '[data-testid="policy-criteria-value-number-input"]',
                select: '[data-testid="policy-criteria-value-select"]',
                selectOption: '[data-testid="policy-criteria-value-select-option"]',
                radioGroup: '[data-testid="policy-criteria-value-radio-group"]',
                radioGroupItem: '[data-testid="policy-criteria-value-radio-group-item"]',
                radioGroupString: '[data-testid="policy-criteria-value-radio-group-string"]',
                radioGroupStringItem:
                    '[data-testid="policy-criteria-value-radio-group-string-item"]',
                negateCheckbox: '[data-testid="policy-criteria-value-negate-checkbox"]',
                multiselect: '[data-testid="policy-criteria-value-multiselect"]',
                multiselectOption: '[data-testid="policy-criteria-value-multiselect-option"]',
                tableModal: {
                    textInput: '[data-testid="table-modal-text-input"]',
                    openButton: '[data-testid="table-modal-open-button"]',
                    firstRowName: '[data-testid="table-modal-table"] tbody tr:nth(0) a',
                    firstRowCheckbox:
                        '[data-testid="table-modal-table"] tbody .pf-c-table__check input[type="checkbox"]',
                    saveBtn: '[data-testid="table-modal-save-btn"]',
                    cancelBtn: '[data-testid="table-modal-cancel-btn"]',
                    emptyState: '[data-testid="table-modal-empty-state"]',
                },
                addBtn: '[data-testid="add-policy-criteria-value-btn"]',
                deleteBtn: '[data-testid="delete-policy-criteria-value-btn"]',
            },
            deleteBtn: '[data-testid="delete-policy-criteria-btn"]',
            booleanOperator: '[data-testid="policy-criteria-boolean-operator"]',
        },
    },
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
    confirmationModal: scopeSelectors('[aria-label="Confirm delete"]', {
        deleteButton: 'button:contains("Delete")',
    }),
};
