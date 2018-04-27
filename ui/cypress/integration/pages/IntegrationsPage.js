export const url = '/main/integrations';

export const selectors = {
    configure: 'nav.left-navigation li:contains("Configure") a',
    navLink: '.navigation-panel li:contains("Integrations") a',
    dockerSwarmTile: 'button:contains("Docker Swarm")',
    clusters: {
        swarmCluster1: 'tr:contains("Swarm Cluster 1")'
    },
    buttons: {
        addCluster: 'button:contains("Add")',
        next: 'button:contains("Next")',
        download: 'button:contains("Download")'
    },
    form: {
        cluster: {
            inputs: '.cluster-form input:not(:last)',
            checkbox: '.cluster-form input:last'
        }
    },
    readOnlyView: '.overflow-auto > .p-4 > div',
    plugins: '.mb-6:first button'
};
