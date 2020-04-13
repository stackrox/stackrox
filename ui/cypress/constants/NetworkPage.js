export const url = '/main/network';

export const selectors = {
    network: 'nav.left-navigation li:contains("Network") a',
    simulatorSuccessMessage: 'div:contains("Policies processed")',
    panels: {
        creatorPanel: '[data-testid="network-creator-panel"]',
        simulatorPanel: '[data-testid="network-simulator-panel"]',
        uploadPanel: '[data-testid="upload-yaml-panel"]',
        detailsPanel: '[data-testid="network-details-panel"]'
    },
    legend: {
        deployments: '[data-testid="deployment-legend"] div',
        namespaces: '[data-testid="namespace-legend"] div',
        connections: '[data-testid="connection-legend"] div'
    },
    namespaces: {
        all: 'g.container > rect',
        getNamespace: namespace => `g.container > rect.namespace-${namespace}`
    },
    services: {
        all: 'g.namespace .node',
        getServicesForNamespace: namespace => `g.namespace-${namespace} .node`
    },
    links: {
        all: '.link',
        bidirectional: '.link[marker-start="url(#start)"]',
        namespaces: '.link.namespace',
        services: '.link.service'
    },
    buttons: {
        viewActiveYamlButton: '[data-testid="view-active-yaml-button"]',
        simulatorButtonOn: '[data-testid="simulator-button-on"]',
        simulatorButtonOff: '[data-testid="simulator-button-off"]',
        generateNetworkPolicies: 'button:contains("Generate and simulate network policies")'
    }
};
