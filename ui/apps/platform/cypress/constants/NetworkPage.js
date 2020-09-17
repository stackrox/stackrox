import scopeSelectors from '../helpers/scopeSelectors';
import search from '../selectors/search';

export const url = '/main/network';

const networkPanels = {
    creatorPanel: '[data-testid="network-creator-panel"]',
    simulatorPanel: '[data-testid="network-simulator-panel"]',
    uploadPanel: '[data-testid="upload-yaml-panel"]',
    detailsPanel: '[data-testid="network-details-panel"]',
};

export const selectors = {
    network: 'nav.left-navigation li:contains("Network") a',
    simulatorSuccessMessage: 'div:contains("Policies processed")',
    panels: networkPanels,
    legend: {
        deployments: '[data-testid="deployment-legend"] div',
        namespaces: '[data-testid="namespace-legend"] div',
        connections: '[data-testid="connection-legend"] div',
    },
    namespaces: {
        all: 'g.container > rect',
        getNamespace: (namespace) => `g.container > rect.namespace-${namespace}`,
    },
    services: {
        all: 'g.namespace .node',
        getServicesForNamespace: (namespace) => `g.namespace-${namespace} .node`,
    },
    links: {
        all: '.link',
        bidirectional: '.link[marker-start="url(#start)"]',
        namespaces: '.link.namespace',
        services: '.link.service',
    },
    buttons: {
        viewActiveYamlButton: '[data-testid="view-active-yaml-button"]',
        simulatorButtonOn: '[data-testid="simulator-button-on"]',
        simulatorButtonOff: '[data-testid="simulator-button-off"]',
        generateNetworkPolicies: 'button:contains("Generate and simulate network policies")',
    },
    networkFlowsSearch: scopeSelectors(networkPanels.detailsPanel, {
        ...search,
    }),
    networkDetailsPanel: scopeSelectors(networkPanels.detailsPanel, {
        header: '[data-testid="network-details-panel-header"]',
    }),
};
