export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Platform Configuration") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    newPolicyButton: 'button:contains("New")',
    importPolicyButton: 'button[data-testid="import-policy-btn"]:contains("Import Policy")',
    singlePolicyExportButton: 'button[data-testid="single-policy-export"]',
    editPolicyButton: 'button:contains("Edit")',
    savePolicyButton: 'button:contains("Save")',
    nextButton: '.btn:contains("Next")',
    prevButton: 'button:contains("Previous")',
    cancelButton: 'button[data-testid="cancel"]',
    reassessAllButton: 'button:contains("Reassess")',
    actionMenuBtn: 'button:contains("Actions")',
    actionMenu: '[data-testid="menu-list"]',
    checkboxes: 'input:checkbox',
    policies: {
        scanImage: 'div.rt-tr:contains("90-Day")',
        addCapabilities: '.rt-tr:contains("CAP_SYS_ADMIN capability added")',
        disabledPolicyImage: 'div.rt-tr.data-test-disabled:first',
    },
    form: {
        nameInput: 'form input[name=name]',
        enableField: 'form input[name=disabled]',
        required: 'form span[data-testid="required"]',
        select: 'form select',
        selectValue: 'form .react-select__multi-value__label',
    },
    configurationField: {
        select: '#policyConfigurationSelect',
        selectArrow: '#policyConfigurationSelect .react-select__dropdown-indicator',
        options: '#policyConfigurationSelect .react-select__option',
        numericInput: '[data-testid="policyConfigurationFields"] .react-numeric-input input',
    },
    imageRegistry: {
        input: 'input[name="fields.imageName.registry"]',
        deleteButton: 'div:contains("Image Registry")+ div.flex>div.flex>button',
        value: '[data-testid="imageName"] div.flex',
    },
    scanAgeDays: {
        input:
            'div:contains("Days since image was last scanned") + div.flex>.react-numeric-input>input',
        deleteButton:
            'div:contains("Days since image was last scanned") + div.flex>div.flex>button',
        value: '[data-testid="scanAgeDays"] div.flex',
    },
    categoriesField: {
        input: 'div:contains("Categories") + div.flex .react-select__input > input',
        valueContainer: 'div:contains("Categories") + div.flex .react-select__value-container',
    },
    policyPreview: {
        loading: '[data-testid="dry-run-loading"]',
        message: '.warn-message',
        alertPreview: {
            table: '.alert-preview table',
        },
    },
    policyDetailsPanel: {
        detailsSection: '[data-testid="policy-details"]',
        criteriaSection: '[data-testid="policy-criteria"]',
        idValueDiv: 'div.text-base-600:contains("ID:") + div',
        enabledValueDiv: 'div.text-base-600:contains("Enabled") + div',
    },
    policyImportModal: {
        content: '[data-testid="policy-import-modal-content"]',
        uploadIcon: '[data-testid="policy-import-modal-content"] [data-testid="upload-icon"]',
        fileInput: '[data-testid="policy-import-modal-content"] input[type="file"]',
        policyNames: '[data-testid="policies-to-import"] li',
        cancel: '[data-testid="custom-modal-cancel"]',
        confirm: '[data-testid="custom-modal-confirm"]',
        imports: '[data-testid="policies-to-import"]',
        successMessage:
            '[data-testid="policy-import-modal-content"] [data-testid="message"].info-message:contains("Policy successfully imported")',
        dupeNameMessage:
            '[data-testid="policy-import-modal-content"] [data-testid="message"].error-message:contains("An existing policy has the same name")',
        dupeIdMessage:
            '[data-testid="policy-import-modal-content"] [data-testid="message"].error-message:contains("has the same ID")',
        renameRadioLabel:
            '[data-testid="dupe-policy-form"] label:contains("Rename incoming policy")',
        overwriteRadioLabel:
            '[data-testid="dupe-policy-form"] label:contains("Overwrite existing policy")',
        keepBothRadioLabel:
            '[data-testid="dupe-policy-form"] label:contains("Keep both policies (imported policy will be assigned a new ID)")',
        newNameInputLabel: '[data-testid="dupe-policy-form"] label:contains("New name")',
    },
    searchInput: '.react-select__input > input',
    sidePanel: '[data-testid="side-panel"]',
    sidePanelHeader: '[data-testid="side-panel-header"]',
    tableFirstRow: 'div.rt-tbody > div.rt-tr-group:first > .rt-tr.-odd',
    tableFirstRowName:
        'div.rt-tbody > div.rt-tr-group:first > .rt-tr.-odd [data-testid=policy-name]',
    hoverActionButtons: '.rt-tr-actions svg',
    tableContainer: '[data-testid="policies-table-container"]',
    enableDisableIcon: '[data-testid="enable-disable-icon"]',
    enabledIconColor: 'bg-success-500',
    enforcement: {
        buildTile: '[data-testid="policy-enforcement-build-tile"]',
        deployTile: '[data-testid="policy-enforcement-deploy-tile"]',
        onOffToggle: '[data-testid="policy-enforcement-on-off"]',
    },
    toast: '.toast-selector',
    booleanPolicySection: {
        addPolicySectionBtn: '[data-testid="add-policy-section-btn"]',
        policySection: '[data-testid="policy-section"]',
        sectionHeader: {
            text: '[data-testid="section-header"]',
            input: '[data-testid="section-header"] input',
            editBtn: '[data-testid="section-header-edit-btn"]',
            confirmBtn: '[data-testid="section-header-confirm-btn"]',
        },
        form: {
            selectArrow: '[data-testid="policy-field-value"] .react-select__dropdown-indicator',
            selectOption: '[data-testid="policy-field-value"] .react-select__option',
            numericInput: '[data-testid="policy-field-value"]  .react-numeric-input input',
            textInput: '[data-testid="policy-field-value"] input',
        },
        removePolicySectionBtn: '[data-testid="remove-policy-section-btn"]',
        policyKey: '[data-testid="draggable-policy-key"]',
        policyKeyGroupBtn: '[data-testid="policy-key-group"] [data-testid="collapsible-btn"]',
        policyKeyGroupContent:
            '[data-testid="policy-key-group"] [data-testid="collapsible-content"]',
        policySectionDropTarget: '[data-testid="policy-section-drop-target"]',
        policyFieldCard: '[data-testid="policy-field-card"]',
        policyFieldValue: '[data-testid="policy-field-value"]',
        andOrOperator: '[data-testid="and-or-operator"]',
        notToggle: '[data-testid="not-toggle"]',
        removePolicyFieldBtn: '[data-testid="remove-policy-field-card-btn"]',
        addPolicyFieldValueBtn: '[data-testid="add-policy-field-value-btn"]',
        removePolicyFieldValueBtn: '[data-testid="remove-policy-field-value-btn"]',
    },
};

export const text = {
    policyLatestTagName: 'Latest tag',
    policyPreview: {
        message:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.',
    },
};
