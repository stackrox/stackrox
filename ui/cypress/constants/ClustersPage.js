export const clustersUrl = '/main/clusters';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Clusters") a',
    header: '[data-test-id="header-text"]',
    autoUpgradeInput: '[id="enableAutoUpgrade"]',
    clusters: {
        k8sCluster0: 'div.rt-td:contains("Kubernetes Cluster 0")'
    },
    buttons: {
        new: 'button:contains("New")',
        next: 'button:contains("Next")',
        downloadYAML: 'button:contains("Download YAML")',
        delete: 'button:contains("Delete")',
        test: 'button:contains("Test")',
        create: 'button:contains("Create")',
        confirmDelete: '.dialog button:contains("Delete")',
        generate: 'button:contains("Generate"):not([disabled])',
        revoke: 'button:contains("Revoke")',
        closePanel: 'button[data-test-id="cancel"]'
    },
    clusterForm: {
        nameInput: 'form[data-testid="cluster-form"] input[name="name"]',
        imageInput: 'form[data-testid="cluster-form"] input[name="mainImage"]',
        endpointInput: 'form[data-testid="cluster-form"] input[name="centralApiEndpoint"]'
    },
    dialog: '.dialog',
    checkboxes: 'input[data-testid="checkbox-table-row-selector"'
};
