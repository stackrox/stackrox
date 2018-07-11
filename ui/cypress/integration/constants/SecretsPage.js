export const url = '/main/secrets';

export const selectors = {
    secrets: 'nav.left-navigation li:contains("Secrets") a',
    panel: {
        secretDetails: 'div:contains("Overview")'
    },
    cancelButton: 'button[data-test-id="cancel"]',
    deploymentLink: 'div:contains("Deployment Name") + a',
    table: {
        rows: 'table tr.cursor-pointer'
    },
    searchInput: '.Select-input > input'
};
