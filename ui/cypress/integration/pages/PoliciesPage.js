export const url = '/main/policies';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("System Policies") a',
    addPolicyButton: 'button:contains("Add")',
    editPolicyButton: 'button:contains("Edit")',
    savePolicyButton: 'button:contains("Save")',
    nextButton: 'button:contains("Next")',
    prevButton: 'button:contains("Previous")',
    cancelButton: 'button.cancel',
    policies: {
        latest: 'tbody > tr:contains("latest")'
    },
    form: {
        disabled: 'form #disabled'
    },
    policyPreview: {
        message: '.warn-message',
        alertPreview: {
            table: '.alert-preview table'
        }
    }
};

export const text = {
    policyPreview: {
        message:
            'This policy is not currently enabled. If enabled, the policy would generate violations for the following deployments on your system.'
    }
};
