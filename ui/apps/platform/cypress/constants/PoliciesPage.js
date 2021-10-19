import navigationSelectors from '../selectors/navigation';

export const url = '/main/policies';

export const selectors = {
    configure: `${navigationSelectors.navExpandable}:contains("Platform Configuration")`,
    navLink: `${navigationSelectors.nestedNavLinks}:contains("System Policies")`,
    newPolicyButton: 'button:contains("New")',
    importPolicyButton: 'button[data-testid="import-policy-btn"]:contains("Import Policy")',
    singlePolicyExportButton: 'button[data-testid="single-policy-export"]',
    clonePolicyButton: 'button:contains("Clone")',
    editPolicyButton: 'button:contains("Edit")',
    savePolicyButton: 'button:contains("Save")',
    nextButton: '.btn:contains("Next")',
    prevButton: 'button:contains("Previous")',
    cancelButton: 'button[data-testid="cancel"]',
    reassessAllButton: 'button:contains("Reassess")',
    actionsButton: 'button[data-testid="menu-button"]:contains("Actions")',
    actionsMenuButton: '[data-testid="menu-list"] button',
    checkbox1: '.rt-thead input:checkbox',
    policies: {
        disabledPolicyImage: 'div.rt-tr.data-test-disabled:first [data-testid=policy-name]',
    },
    form: {
        nameInput: 'form input[name=name]',
        enableField: 'form input[name=disabled]',
        required: 'span[data-testid="required"]',
        select: 'form select',
        selectValue: 'form .react-select__single-value',
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
        input: 'div:contains("Days since image was last scanned") + div.flex>.react-numeric-input>input',
        deleteButton:
            'div:contains("Days since image was last scanned") + div.flex>div.flex>button',
        value: '[data-testid="scanAgeDays"] div.flex',
    },
    imageUser: {
        input: 'div:contains("imageUser input json") + div.flex>.react-numeric-input>input',
        deleteButton: 'div:contains("Image User") + div.flex>div.flex>button',
        value: '[data-testid="imageUser"] div.flex',
    },
    categoriesField: {
        input: 'div:contains("Categories") + div.flex .react-select__input > input',
        valueContainer: 'div:contains("Categories") + div.flex .react-select__value-container',
    },
    lifecycleStageField: {
        select: '[data-testid="lifecycle-stages"] div:nth(0)',
        input: '[data-testid="lifecycle-stages"] .react-select__input > input',
        clearBtn: '[data-testid="lifecycle-stages"] div.react-select__clear-indicator',
    },
    eventSourceField: {
        select: '[data-testid="event-sources"] div:nth(0)',
        selectArrow: '[data-testid="event-sources"] .react-select__dropdown-indicator',
        options: '[data-testid="event-sources"] .react-select__option',
    },
    restrictToScopeField: {
        addBtn: '[data-testid="restrict-to-scope"] [data-testid="add-scope"]',
        labelKeyInput: '[data-testid="restrict-to-scope"] [name="scope[0].label.key"]',
        labelValueInput: '[data-testid="restrict-to-scope"] [name="scope[0].label.value"]',
    },
    excludeByScopeField: {
        addBtn: '[data-testid="exclude-by-scope"] [data-testid="add-scope"]',
        labelKeyInput:
            '[data-testid="exclude-by-scope"] [name="whitelistedDeploymentScopes[0].scope.label.key"]',
        labelValueInput:
            '[data-testid="exclude-by-scope"] [name="whitelistedDeploymentScopes[0].scope.label.value"]',
        deploymentNameSelect: '[data-testid="exclude-by-scope"] .react-select__control:nth(1)',
    },
    excludedImagesField: {
        select: '[data-testid="excluded-images"] div:nth(0)',
        input: '[data-testid="excluded-images"] .react-select__input > input',
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
            '[data-testid="policy-import-modal-content"] [data-testid="message"].success-message:contains("Policy successfully imported")',
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
    tableRowName: 'div.rt-tbody [data-testid="policy-name"]',
    hoverActionButtons: '.rt-tr-actions svg',
    tableContainer: '[data-testid="policies-table-container"]',
    deleteButton: '.rt-tr-actions div button:nth-child(2)',
    enableDisableButton: '.rt-tr-actions div button:nth-child(1)',
    enableDisableIcon: '[data-testid="enable-disable-icon"]',
    enabledIconColor: 'bg-success-500',
    disabledIconColor: 'bg-base-300',
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
        policyKeyGroup: '[data-testid="policy-key-group"]',
        policyKeyFilter: '[data-testid="policy-key-filter"]',
        collapsibleBtn: '[data-testid="collapsible-btn"]',
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
    addCapabilities: 'CAP_SYS_ADMIN capability added',
    policyLatestTagName: 'Latest tag',
    policyPreview: {
        disabled:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.',
    },
    scanImage: '90-Day',
};
