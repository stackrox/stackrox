export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    newPolicyButton: 'button:contains("New")',
    editPolicyButton: 'button:contains("Edit")',
    savePolicyButton: 'button:contains("Save")',
    nextButton: '.btn:contains("Next")',
    prevButton: 'button:contains("Previous")',
    cancelButton: 'button[data-test-id="cancel"]',
    reassessAllButton: 'button:contains("Reassess")',
    deleteButton: 'button:contains("Delete")',
    checkboxes: 'input:checkbox',
    policies: {
        scanImage: 'div.rt-tr:contains("90-Day")',
        addCapabilities: '.rt-tr:contains("CAP_SYS_ADMIN capability added")'
    },
    form: {
        nameInput: 'form input:first',
        enableField: 'form div:contains("Enable") + div',
        required: 'form span[data-test-id="required"]',
        select: 'form select',
        selectValue: 'form .react-select__multi-value__label'
    },
    configurationField: {
        select: 'form [data-test-id="policyConfiguration"] select',
        selectArrow:
            '[data-test-id="policyConfiguration"] .react-select__control .react-select__dropdown-indicator',
        options: '[data-test-id="policyConfiguration"] div[role="option"]',
        numericInput: '[data-test-id="policyConfiguration"] .react-numeric-input input'
    },
    imageRegistry: {
        input: 'input[name="fields.imageName.registry"]',
        deleteButton: 'div:contains("Image Registry")+ div.flex>div.flex>button',
        value: '[data-test-id="imageName"] div.flex'
    },
    scanAgeDays: {
        input: 'div:contains("Days since Image scanned") + div.flex>.react-numeric-input>input',
        deleteButton: 'div:contains("Days since Image scanned") + div.flex>div.flex>button',
        value: '[data-test-id="scanAgeDays"] div.flex'
    },
    policyPreview: {
        message: '.warn-message',
        alertPreview: {
            table: '.alert-preview table'
        }
    },
    policyDetailsPanel: {
        idValueDiv: 'div.text-base-600:contains("Id") + div',
        enabledValueDiv: 'div.text-base-600:contains("Enabled") + div'
    },
    searchInput: '.react-select__input > input',
    sidePanel: '[data-test-id="panel"]',
    sidePanelHeader: '[data-test-id="panel-header"]',
    tableFirstRow: 'div.rt-tbody > div.rt-tr-group:first > .rt-tr.-odd',
    hoverActionButtons: '.rt-tr-actions svg',
    tableContainer: '[data-test-id="policies-table-container"]',
    enableDisableIcon: '[data-test-id="enable-disable-icon"]',
    enabledIconColor: 'bg-success-500'
};

export const text = {
    policyLatestTagName: 'Latest tag',
    policyPreview: {
        message:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.'
    }
};
