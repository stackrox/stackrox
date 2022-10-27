import * as api from '../constants/apiEndpoints';
import { selectors as networkGraphSelectors } from '../constants/NetworkPage';
import { visitFromLeftNav } from './nav';
import { interactAndWaitForResponses } from './request';
import { visit } from './visit';
import selectSelectors from '../selectors/select';
import tabSelectors from '../selectors/tab';

const getNodeErrorMessage = (node) => `Could not find node "${node.name}" of type "${node.type}"`;

const getEdgeErrorMessage = (sourceNode, targetNode) =>
    `Could not find an edge between "${sourceNode.name}" and "${targetNode.name}"`;

const getEdgePresentErrorMessage = (sourceNode, targetNode) =>
    `Found an edge between "${sourceNode.name}" and "${targetNode.name}" when there wasn't supposed to be one`;

// Network Graph Interaction-based Commands

export function clickOnNodeById(cytoscape, node) {
    const element = cytoscape.getElementById(node.id);
    if (!element) {
        throw Error(getNodeErrorMessage(node));
    }
    element.emit('click');
}

export function clickOnNodeByName(cytoscape, node) {
    const filteredNodes = cytoscape.nodes().filter(filterByNodeName(node));
    if (filteredNodes.length === 0) {
        throw Error(getNodeErrorMessage(node));
    }
    filteredNodes.emit('click');
}

export const networkBaselineStatusAlias = 'networkbaseline/id/status';

const requestConfigForDeploymentNode = {
    routeMatcherMap: {
        [networkBaselineStatusAlias]: {
            method: 'POST',
            url: api.network.networkBaselineStatus,
        },
    },
};

export function clickOnDeploymentNodeByName(cytoscape, name) {
    interactAndWaitForResponses(() => {
        clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name });
    }, requestConfigForDeploymentNode);
}

export function mouseOverNodeById(cytoscape, node) {
    const element = cytoscape.getElementById(node.id);
    if (!element) {
        throw Error(getNodeErrorMessage(node));
    }
    element.emit('mouseover');
}

export function mouseOverNodeByName(cytoscape, node) {
    const filteredNodes = cytoscape.nodes().filter(filterByNodeName(node));
    if (filteredNodes.length === 0) {
        throw Error(getNodeErrorMessage(node));
    }
    filteredNodes.emit('mouseover');
}

export function mouseOverEdgeByNames(cytoscape, sourceNode, targetNode) {
    const edges = cytoscape.edges().filter(filterBySourceTarget(sourceNode, targetNode));
    if (edges.length === 0) {
        throw Error(getEdgeErrorMessage(sourceNode, targetNode));
    }
    edges.emit('mouseover');
}

export function ensureEdgeNotPresent(cytoscape, sourceNode, targetNode) {
    const edges = cytoscape.edges().filter(filterBySourceTarget(sourceNode, targetNode));
    if (edges.length !== 0) {
        throw Error(getEdgePresentErrorMessage(sourceNode, targetNode));
    }
}

// Filter Functions

export function filterDeployments(element) {
    return element.data('type') === 'DEPLOYMENT';
}

export function filterNamespaces(element) {
    return element.data('type') === 'NAMESPACE';
}

export function filterClusters(element) {
    return element.data('type') === 'CLUSTER';
}
export function filterInternet(element) {
    return element.data('type') === 'INTERNET';
}

export function filterByNodeName(node) {
    return (element) => {
        return element.data('type') === node.type && element.data('name') === node.name;
    };
}

export function filterBySourceTarget(sourceNode, targetNode) {
    return (element) => {
        if (sourceNode.type === 'DEPLOYMENT' && targetNode.type === 'DEPLOYMENT') {
            return (
                element.data('type') === 'NODE_TO_NODE_EDGE' &&
                element.data('sourceNodeName') === sourceNode.name &&
                element.data('targetNodeName') === targetNode.name
            );
        }
        if (sourceNode.type === 'NAMESPACE' && targetNode.type === 'NAMESPACE') {
            return (
                element.data('type') === 'NAMESPACE_EDGE' &&
                element.data('sourceNodeNamespace') === sourceNode.name &&
                element.data('targetNodeNamespace') === targetNode.name
            );
        }
        if (sourceNode.type === 'DEPLOYMENT' && targetNode.type === 'NAMESPACE') {
            return (
                element.data('type') === 'NODE_TO_NAMESPACE_EDGE' &&
                element.data('sourceNodeName') === sourceNode.name &&
                element.data('target').startsWith(targetNode.name)
            );
        }
        if (sourceNode.type === 'NAMESPACE' && targetNode.type === 'DEPLOYMENT') {
            return (
                element.data('type') === 'NODE_TO_NAMESPACE_EDGE' &&
                element.data('source').startsWith(sourceNode.name) &&
                element.data('targetNodeName') === targetNode.name
            );
        }
        throw Error(
            `An edge type between a (${sourceNode.type}) and (${targetNode.type}) does not exist`
        );
    };
}

// search filters

const networkGraphClusterAlias = 'networkgraph/cluster/id';
const networkPoliciesClusterAlias = 'networkpolicies/cluster/id';

const requestConfigToFilterGraph = {
    routeMatcherMap: {
        [networkGraphClusterAlias]: {
            method: 'GET',
            url: api.network.networkGraph,
        },
        [networkPoliciesClusterAlias]: {
            method: 'GET',
            url: api.network.networkPoliciesGraph,
        },
    },
};

export function selectDeploymentFilter(deploymentName) {
    interactAndWaitForResponses(() => {
        cy.get(networkGraphSelectors.toolbar.filterSelect).type('Deployment{enter}');
        cy.get(networkGraphSelectors.toolbar.filterSelect).type(`${deploymentName}{enter}{esc}`);
    }, requestConfigToFilterGraph);
}

// Additional calls in a test can select additional namespaces.

export function selectNamespaceFilter(namespace) {
    interactAndWaitForResponses(() => {
        cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
        cy.get(
            `${selectSelectors.patternFlySelect.openMenu} span:contains("${namespace}")`
        ).click();
        cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
    }, requestConfigToFilterGraph);
}

export function selectNamespaceFilterWithGraphAndPoliciesFixtures(
    namespace,
    fixturePathGraph,
    fixturePathPolicies
) {
    interactAndWaitForResponses(
        () => {
            cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
            cy.get(
                `${selectSelectors.patternFlySelect.openMenu} span:contains("${namespace}")`
            ).click();
            cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
        },
        requestConfigToFilterGraph,
        {
            [networkGraphClusterAlias]: { fixture: fixturePathGraph },
            [networkPoliciesClusterAlias]: { fixture: fixturePathPolicies },
        }
    );
}

export function selectNamespaceFilterWithNetworkGraphResponse(namespace, response) {
    cy.intercept('GET', api.network.networkGraph, response).as('networkGraph');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('networkPolicies');

    interactAndWaitForResponses(
        () => {
            cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
            cy.get(
                `${selectSelectors.patternFlySelect.openMenu} span:contains("${namespace}")`
            ).click();
            cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
        },
        requestConfigToFilterGraph,
        {
            [networkGraphClusterAlias]: response,
        }
    );
}

// visit helpers

export const notifiersAlias = 'notifiers';
export const clustersAlias = 'clusters';
export const networkPoliciesGraphEpochAlias = 'networkpolicies/graph/epoch';
export const searchMetadataOptionsAlias = 'search/metadata/options';
export const getClusterNamespaceNamesAlias = 'getClusterNamespaceNames';

const requestConfigToVisitGraph = {
    routeMatcherMap: {
        [notifiersAlias]: {
            method: 'GET',
            url: api.integrations.notifiers,
        },
        [clustersAlias]: {
            method: 'GET',
            url: api.clusters.list,
        },
        [networkPoliciesGraphEpochAlias]: {
            method: 'GET',
            url: `${api.network.epoch}?clusterId=null`,
        },
        [searchMetadataOptionsAlias]: {
            method: 'GET',
            url: api.search.optionsCategories('DEPLOYMENTS'),
        },
        // Network Graph makes the following query on the first visit, but not subsequent visit via browser Back button.
        // Include it because each cypress test has a new connection, therefore behaves as a first visit.
        [getClusterNamespaceNamesAlias]: {
            method: 'POST',
            url: api.graphql('getClusterNamespaceNames'),
        },
    },
};

export const basePath = '/main/network';

/*
 * Reach clusters by interaction from another container.
 * For example, click View Deployment in Network Graph button from Risk.
 */
export function reachNetworkGraph(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(interactionCallback, requestConfigToVisitGraph, staticResponseMap);

    cy.location('pathname').should('contain', basePath); // contain because pathname might have id
    cy.get(networkGraphSelectors.networkGraphHeading);
}

export function visitNetworkGraphFromLeftNav() {
    visitFromLeftNav('Network', requestConfigToVisitGraph);

    cy.location('pathname').should('eq', basePath);
    cy.get(networkGraphSelectors.networkGraphHeading);
    cy.get(networkGraphSelectors.emptyStateSubheading);
}

export function visitNetworkGraph(staticResponseMap) {
    visit(basePath, requestConfigToVisitGraph, staticResponseMap);

    cy.get(networkGraphSelectors.networkGraphHeading);
    cy.get(networkGraphSelectors.emptyStateSubheading);
}

export function visitNetworkGraphWithNamespaceFilter(namespace) {
    visitNetworkGraph();
    selectNamespaceFilter(namespace);
}

export function visitNetworkGraphWithMockedData() {
    visitNetworkGraph();
    selectNamespaceFilterWithGraphAndPoliciesFixtures(
        'stackrox',
        'network/networkGraph.json',
        'network/networkPolicies.json'
    );
}

// deployment Network Flows

export const networkBaselinePeersAlias = 'networkbaseline/id/peers';

export function interactAndWaitForChangeToNetworkFlows(interactionCallback) {
    interactAndWaitForResponses(interactionCallback, {
        routeMatcherMap: {
            [networkBaselinePeersAlias]: {
                method: 'PATCH',
                url: api.network.networkBaselinePeers,
            },
            [networkGraphClusterAlias]: {
                method: 'GET',
                url: api.network.networkGraph,
            },
            [networkPoliciesClusterAlias]: {
                method: 'GET',
                url: api.network.networkPoliciesGraph,
            },
            [networkBaselineStatusAlias]: {
                method: 'POST',
                url: api.network.networkBaselineStatus,
            },
        },
    });
}

// deployment Baseline Settings

export const networkBaselineAlias = 'networkbaseline/id';

export function clickBaselineSettingsTab() {
    interactAndWaitForResponses(
        () => {
            cy.get(`${tabSelectors.tabs}:contains('Baseline Settings')`).click();
        },
        {
            routeMatcherMap: {
                [networkBaselineAlias]: {
                    method: 'GET',
                    url: api.network.networkBaseline,
                },
            },
        }
    );
}
