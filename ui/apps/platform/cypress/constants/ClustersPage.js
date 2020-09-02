import scopeSelectors from '../helpers/scopeSelectors';

export const clustersUrl = '/main/clusters';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Platform Configuration") a',
    navLink: '.navigation-panel li:contains("Clusters") a',
    header: '[data-testid="header-text"]',
    autoUpgradeInput: '[id="enableAutoUpgrade"]',
    clusters: scopeSelectors('[data-testid="clusters-table"]', {
        // Ignore the first checkbox column and last delete column.
        tableHeadingCell: '.rt-th:not(:first-child):not(.hidden)',
        tableDataCell: '.rt-tr-group:not(.hidden) .rt-td:not(:first-child):not(.hidden)',
        tableRowGroup: '.rt-tr-group:not(.hidden)',
        k8sCluster0: 'div.rt-td:contains("Kubernetes Cluster 0")',
    }),
    buttons: {
        new: 'button:contains("New")',
        next: 'button:contains("Next")',
        downloadYAML: 'button:contains("Download YAML")',
        delete: 'button:contains("Delete")',
        test: 'button:contains("Test")',
        create: 'button:contains("Create")',
        cancelDelete: '.dialog button:contains("Cancel")',
        confirmDelete: '.dialog button:contains("Delete")',
        generate: 'button:contains("Generate"):not([disabled])',
        revoke: 'button:contains("Revoke")',
        closePanel: 'button[data-testid="cancel"]',
    },
    clusterForm: scopeSelectors('[data-testid="cluster-form"]', {
        nameInput: 'input[name="name"]',
        imageInput: 'input[name="mainImage"]',
        endpointInput: 'input[name="centralApiEndpoint"]',
    }),
    clusterHealth: scopeSelectors('[data-testid="clusters-side-panel"]', {
        clusterStatus: '[data-testid="clusterStatus"]',
        sensorStatus: '[data-testid="sensorStatus"]',
        collectorStatus: '[data-testid="collectorStatus"]',
        totalReadyPods: '[data-testid="totalReadyPods"]',
        totalDesiredPods: '[data-testid="totalDesiredPods"]',
        totalRegisteredNodes: '[data-testid="totalRegisteredNodes"]',
        healthInfoComplete: '[data-testid="healthInfoComplete"]',
        sensorUpgrade: '[data-testid="sensorUpgrade"]',
        sensorVersion: '[data-testid="sensorVersion"]',
        centralVersion: '[data-testid="centralVersion"]',
        credentialExpiration: '[data-testid="credentialExpiration"]',
    }),
    dialog: '.dialog',
    checkboxes: 'input[data-testid="checkbox-table-row-selector"',
    sidePanel: '[data-testid="clusters-side-panel"]',
    credentialExpirationBanner: '[data-testid="credential-expiration-banner"]',
};
