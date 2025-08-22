import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/policy-management/policies';

export const selectors = {
    table: {
        createButton: 'button:contains("Create policy")',
        importButton: 'button:contains("Import policy")',
        searchInput: '.react-select__input > input',
        bulkActionsDropdownButton: 'button:contains("Bulk actions")',
        bulkActionsDropdownItem: 'button:contains("Bulk actions") + div ul[role="menu"] li',
        reassessButton: 'button:contains("Reassess all")',
        policyLink: 'td[data-label="Policy"] a',
        statusCell: 'td[data-label="Status"]',
        originCell: 'td[data-label="Origin"]',
        severityCell: 'td[data-label="Severity"]',
        lifecycleCell: 'td[data-label="Lifecycle"]',
        selectCheckbox: '.pf-v5-c-table__check input[type="checkbox"]',
        actionsToggleButton: 'td.pf-v5-c-table__action button',
        actionsItemButton: 'td.pf-v5-c-table__action ul li button[role="menuitem"]',
        firstRow: '[data-testid="policies-table"] tbody tr:nth(0)',
        rows: '[data-testid="policies-table"] tbody tr',
    },
    page: {
        actionsToggleButton: 'button.pf-v5-c-menu-toggle:contains("Actions")',
        actionsItemButton:
            'button.pf-v5-c-menu-toggle:contains("Actions") + div ul li button[role="menuitem"]',
    },
    toast: {
        title: 'ul.pf-v5-c-alert-group .pf-v5-c-alert__title',
        description: 'ul.pf-v5-c-alert-group .pf-v5-c-alert__description',
    },
    wizardBtns: {
        step3: '.pf-v5-c-wizard__nav-link:contains("Rules")',
        step6: '.pf-v5-c-wizard__nav-link:contains("Review")',
    },
    step3: {
        defaultPolicyAlert: '[data-testid="default-policy-alert"]',
        policySection: {
            addBtn: '[data-testid="add-section-btn"]',
            deleteBtn: '[title="Delete policy section"]',
            cards: '.policy-section-card-body',
            name: '[data-testid="policy-section-name"]',
            nameEditBtn: '.policy-section-card-header [title="Edit name of policy section"]',
            nameSaveBtn: '.policy-section-card-header [title="Save name of policy section"]',
            nameInput: '.policy-section-card-header input',
            orDivider: 'div.or-divider-container',
            dropTarget: '[data-testid="policy-section-drop-target"]',
        },
        policyCriteria: {
            keyGroup: '[data-testid="policy-criteria-key-group"]',
            key: '.policy-criteria-key',
            groupCards: '[data-testid="policy-criteria-group-card"]',
            groupCardTitle:
                '[data-testid="policy-criteria-group-card"] .pf-v5-c-card__header .pf-v5-c-card__title',
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
                        '[data-testid="table-modal-table"] tbody .pf-v5-c-table__check input[type="checkbox"]',
                    saveBtn: '[data-testid="table-modal-save-btn"]',
                    cancelBtn: '[data-testid="table-modal-cancel-btn"]',
                    emptyState: '[data-testid="table-modal-empty-state"]',
                },
                addBtn: '[title="Add value of policy field"]',
                deleteBtn: '[title="Delete value of policy field"]',
            },
            deleteBtn: '[title="Delete policy field"]',
            booleanOperator: '[data-testid="policy-criteria-boolean-operator"]',
        },
    },
    importUploadModal: {
        modalWrapper: '.pf-v5-c-modal-box',
        titleText: '.pf-v5-c-modal-box__title-text:contains("Import policy JSON")',
        fileInput: '.pf-v5-c-file-upload input[type="file"]',
        policyNames: '[data-testid="policies-to-import"] div',
        beginButton: '.pf-v5-c-modal-box__footer button:contains("Begin import")',
        resumeButton: '.pf-v5-c-modal-box__footer button:contains("Resume import")',
        cancelButton: '.pf-v5-c-modal-box__footer button:contains("Cancel")',
        // Form for duplicate policy name
        duplicateAlertTitle:
            '.pf-v5-c-modal-box__body .pf-v5-c-alert__title:contains("Policy already exists")',
        duplicateIdSubstring: '.pf-v5-c-alert__description li:contains("has the same ID")',
        duplicateNameSubstring: '.pf-v5-c-alert__description li:contains("has the same name")',
        keepBothRadioLabel: 'label[for="keep-both-radio"]:contains("Keep both policies")',
        renameRadioLabel: 'label[for="policy-rename-radio"]:contains("Rename incoming policy")',
        renameInput: 'input#policy-rename',
        overwriteRadioLabel:
            'label[for="policy-overwrite-radio-1"]:contains("Overwrite existing policy")',
        errorAlertTitle:
            '.pf-v5-c-modal-box__body .pf-m-danger .pf-v5-c-alert__title:contains("Policy errors causing import failure")',
    },
    importSuccessModal: {
        titleText: '.pf-v5-c-modal-box__title-text:contains("Import policy JSON")',
        policyNames: '[data-testid="policies-imported"] div',
    },
    confirmationModal: scopeSelectors('[aria-label="Confirm delete"]', {
        deleteButton: 'button:contains("Delete")',
    }),
};
