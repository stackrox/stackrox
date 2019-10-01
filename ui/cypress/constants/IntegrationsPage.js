export const url = '/main/integrations';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Integrations") a',
    kubernetesTile: 'div[role="button"]:contains("Kubernetes")',
    dockerRegistryTile: 'div[role="button"]:contains("Generic Docker Registry")',
    clairTile: 'div[role="button"]:contains("CoreOS Clair")',
    clairifyTile: 'div[role="button"]:contains("Clairify")',
    slackTile: 'div[role="button"]:contains("Slack")',
    apiTokenTile: 'div[role="button"]:contains("API Token")',
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
        confirm: 'button:contains("Confirm")',
        generate: 'button:contains("Generate"):not([disabled])',
        revoke: 'button:contains("Revoke")',
        closePanel: 'button[data-test-id="cancel"]'
    },
    apiTokenForm: {
        nameInput: 'form[data-test-id="api-token-form"] input[name="name"]',
        roleSelect: 'form[data-test-id="api-token-form"] .react-select__control'
    },
    apiTokenBox: 'span:contains("eyJ")', // all API tokens start with eyJ
    apiTokenDetailsDiv: 'div[data-test-id="api-token-details"]',
    clusterForm: {
        nameInput: 'form[data-test-id="cluster-form"] input[name="name"]',
        imageInput: 'form[data-test-id="cluster-form"] input[name="mainImage"]',
        endpointInput: 'form[data-test-id="cluster-form"] input[name="centralApiEndpoint"]'
    },
    dockerRegistryForm: {
        nameInput: "form input[name='name']",
        typesSelect: 'form .react-select__control',
        endpointInput: "form input[name='docker.endpoint']"
    },
    labeledValue: '[data-test-id="labeled-value"]',
    plugins: '.mb-6:first div[role="button"]',
    dialog: '.dialog',
    checkboxes: 'input'
};
