import scopeSelectors from '../helpers/scopeSelectors';
import search from '../selectors/search';

export const url = '/main/network';

const networkPanels = {
    creatorPanel: '[data-testid="network-creator-panel"]',
    simulatorPanel: '[data-testid="network-simulator-panel"]',
    uploadPanel: '[data-testid="upload-yaml-panel"]',
    detailsPanel: '[data-testid="network-details-panel"]',
};

const networkEntityTabbedOverlay = '[data-testid="network-entity-tabbed-overlay"]';

export const selectors = {
    cytoscapeContainer: '#cytoscapeContainer',
    networkGraphHeading: 'h1:contains("Network Graph")',
    emptyStateSubheading:
        '.pf-c-empty-state h2:contains("Please select at least one namespace from your cluster")',
    simulatorSuccessMessage: 'div[data-testid="message-body"]:contains("Policies processed")',
    panels: networkPanels,
    legend: {
        deployments: '[data-testid="deployment-legend"]',
        namespaces: '[data-testid="namespace-legend"]',
        connections: '[data-testid="connection-legend"]',
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
        simulatorButtonOff: '[data-testid="simulator-button-off"]',
        generateNetworkPolicies: 'button:contains("Generate and simulate network policies")',
        applyNetworkPolicies: 'button:contains("Apply Network Policies")',
        apply: 'div[aria-modal="true"] button:contains("Apply")',
        // Select buttons by data-testid attribute and contains text, because "allowed" and "all" are ambiguous:
        activeFilter: 'button[data-testid="network-connections-filter-active"]:contains("active")',
        allowedFilter:
            'button[data-testid="network-connections-filter-allowed"]:contains("allowed")',
        allFilter: 'button[data-testid="network-connections-filter-all"]:contains("all")',
        hideNsEdgesFilter: '[data-testid="namespace-flows-filter"] button:contains("Hide")',
        stopSimulation: '.simulator-mode button:contains("Stop")',
        confirmationButton: 'button:contains("Yes")',
    },
    detailsPanel: scopeSelectors(networkPanels.detailsPanel, {
        header: '[data-testid="network-details-panel-header"]',
        search,
        table: {
            rows: '.rt-tbody .rt-tr',
        },
    }),
    networkEntityTabbedOverlay: scopeSelectors(networkEntityTabbedOverlay, {
        header: '[data-testid="network-entity-tabbed-overlay-header"]',
    }),
    toolbar: scopeSelectors('[data-testid="network-graph-toolbar"]', {
        namespaceSelect: '.namespace-select > button',
        filterSelect: search.multiSelectInput,
    }),
    errorOverlay: {
        heading: 'h2:contains("An error has prevented the Network Graph from loading")',
        message: (messageText) =>
            `${selectors.errorOverlay.heading} + div *:contains("${messageText}")`,
    },
};
