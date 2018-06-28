export const url = '/main/integrations';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Integrations") a',
    dockerSwarmTile: 'button:contains("Docker Swarm")',
    kubernetesTile: 'button:contains("Kubernetes")',
    dockerRegistryTile: 'button:contains("Generic Docker Registry")',
    clairTile: 'button:contains("CoreOS Clair")',
    clairifyTile: 'button:contains("Clairify")',
    slackTile: 'button:contains("Slack")',
    clusters: {
        swarmCluster1: 'tr:contains("Swarm Cluster 1")'
    },
    integrationError: 'div[data-test-id="integration-error"',
    buttons: {
        add: 'button:contains("Add")',
        next: 'button:contains("Next")',
        download: 'button:contains("Download")',
        delete: 'button:contains("Delete")',
        test: 'button:contains("Test")',
        create: 'button:contains("Create")',
        confirm: 'button:contains("Confirm")'
    },
    clusterForm: {
        nameInput: 'form[data-test-id="cluster-form"] input[name="name"]',
        imageInput: 'form[data-test-id="cluster-form"] input[name="preventImage"]',
        endpointInput: 'form[data-test-id="cluster-form"] input[name="centralApiEndpoint"]'
    },
    dockerRegistryForm: {
        nameInput: "form label[for='name'] input",
        typesSelect: "form label[for='categories'] div.Select",
        endpointInput: "form label[for='docker.endpoint'] input"
    },
    readOnlyView: '.overflow-auto > .p-4 > div',
    plugins: '.mb-6:first button',
    dialog: '.dialog',
    checkboxes: 'input'
};
