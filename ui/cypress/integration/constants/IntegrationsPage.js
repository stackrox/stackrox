export const url = '/main/integrations';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Integrations") a',
    dockerSwarmTile: 'button:contains("Docker Swarm")',
    kubernetesTile: 'button:contains("Kubernetes")',
    clusters: {
        swarmCluster1: 'tr:contains("Swarm Cluster 1")'
    },
    buttons: {
        addCluster: 'button:contains("Add")',
        next: 'button:contains("Next")',
        download: 'button:contains("Download")',
        delete: 'button:contains("Delete")'
    },
    form: {
        cluster: {
            inputName: ".cluster-form input[name='name']",
            inputImage: ".cluster-form input[name='preventImage']",
            inputEndpoint: ".cluster-form input[name='centralApiEndpoint']"
        }
    },
    readOnlyView: '.overflow-auto > .p-4 > div',
    plugins: '.mb-6:first button',
    dialog: '.dialog',
    checkboxes: 'input'
};
