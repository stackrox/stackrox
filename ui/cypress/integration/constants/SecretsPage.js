export const url = '/main/secrets';

export const selectors = {
    secrets: 'nav.left-navigation li:contains("Secrets") a',
    panel: {
        secretDetails: 'div:contains("Overview")'
    },
    cancelButton: 'button[data-test-id="cancel"]',
    deploymentLinks: 'div[data-test-id="deployments-card"] a',
    table: {
        rows: 'div.rt-tbody div.rt-tr-group'
    },
    searchInput: '.Select-input > input'
};
