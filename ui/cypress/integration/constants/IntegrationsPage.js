export const url = '/main/integrations';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Integrations") a',
    dockerSwarmTile: 'div[role="button"]:contains("Docker Swarm")',
    kubernetesTile: 'div[role="button"]:contains("Kubernetes")',
    dockerRegistryTile: 'div[role="button"]:contains("Generic Docker Registry")',
    clairTile: 'div[role="button"]:contains("CoreOS Clair")',
    clairifyTile: 'div[role="button"]:contains("Clairify")',
    slackTile: 'div[role="button"]:contains("Slack")',
    apiTokenTile: 'div[role="button"]:contains("API Token")',
    clusters: {
        swarmCluster1: 'div.rt-td:contains("Swarm Cluster 1")'
    },
    buttons: {
        new: 'button:contains("New")',
        next: 'button:contains("Next")',
        download: 'button:contains("Download")',
        delete: 'button:contains("Delete")',
        test: 'button:contains("Test")',
        create: 'button:contains("Create")',
        confirm: 'button:contains("Confirm")',
        generate: 'button:contains("Generate"):not([disabled])',
        revoke: 'button:contains("Revoke")'
    },
    apiTokenForm: {
        nameInput: 'form[data-test-id="api-token-form"] input[name="name"]',
        roleSelect: 'form[data-test-id="api-token-form"] div.Select'
    },
    apiTokenBox: 'span:contains("eyJ")', // all API tokens start with eyJ
    apiTokenDetailsDiv: 'div[data-test-id="api-token-details"]',
    clusterForm: {
        nameInput: 'form[data-test-id="cluster-form"] input[name="name"]',
        imageInput: 'form[data-test-id="cluster-form"] input[name="preventImage"]',
        endpointInput: 'form[data-test-id="cluster-form"] input[name="centralApiEndpoint"]'
    },
    dockerRegistryForm: {
        nameInput: "form input[name='name']",
        typesSelect: 'form div.Select',
        endpointInput: "form input[name='docker.endpoint']"
    },
    readOnlyView: '.overflow-auto > .p-4 > div',
    plugins: '.mb-6:first div[role="button"]',
    dialog: '.dialog',
    checkboxes: 'input'
};
