export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    addPolicyButton: 'button:contains("Add")',
    editPolicyButton: 'button:contains("Edit")',
    savePolicyButton: 'button:contains("Save")',
    nextButton: 'button:contains("Next")',
    prevButton: 'button:contains("Previous")',
    cancelButton: 'button[data-test-id="cancel"]',
    policies: {
        latest: 'tbody > tr:contains("latest")'
    },
    form: {
        enableField: 'form div.text-primary-500:contains("Enable") + div',
        required: 'form span[data-test-id="required"]',
        select: 'form select'
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
    policyPreview: {
        message: '.warn-message',
        alertPreview: {
            table: '.alert-preview table'
        }
    },
    policyDetailsPanel: {
        enabledValueDiv: 'div.text-primary-500:contains("Enabled") + div'
    },
    searchInput: '.Select-input > input',
    sidePanel: '[data-test-id="panel"]',
    sidePanelHeader: '[data-test-id="panel-header"]',
    tableFirstRow: 'table tr.cursor-pointer:first',
    enableDisableButton: 'td > button',
    enabledPolicyButtonColorClass: 'text-success-500'
};

export const text = {
    policyPreview: {
        message:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.'
    }
};
