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
    policies: {
        scanImage: 'div.rt-tr:contains("90-Day")',
        addCapabilities: '.rt-tr:contains("CAP_SYS_ADMIN capability added")'
    },
    form: {
        nameInput: 'form input:first',
        enableField: 'form div.text-primary-500:contains("Enable") + div',
        required: 'form span[data-test-id="required"]',
        select: 'form select',
        selectValue: 'form .Select-value-label'
    },
    configurationField: {
        select: 'form [data-test-id="policyConfiguration"] select',
        selectArrow: '[data-test-id="policyConfiguration"] div.Select .Select-arrow',
        options: '[data-test-id="policyConfiguration"] div.Select div[role="option"]',
        numericInput: '[data-test-id="policyConfiguration"] .react-numeric-input input'
    },
    imageRegistry: {
        input: 'input[name="fields.imageName.registry"]',
        deleteButton: 'input[name="fields.imageName.registry"] + div>button',
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
        enabledValueDiv: 'div.text-base-600:contains("Enabled") + div'
    },
    searchInput: '.Select-input > input',
    sidePanel: '[data-test-id="panel"]',
    sidePanelHeader: '[data-test-id="panel-header"]',
    tableFirstRow: 'div.rt-tbody > div.rt-tr-group:first > .rt-tr.-odd',
    tableContainer: '[data-test-id="policies-table-container"]',
    enableDisableButton: '.rt-td > button',
    enabledPolicyButtonColorClass: 'text-success-500'
};

export const text = {
    policyLatestTagName: 'Latest tag',
    policyPreview: {
        message:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.'
    }
};
