import * as api from '../constants/apiEndpoints';
import { selectors as networkGraphSelectors, url as networkUrl } from '../constants/NetworkPage';
import selectSelectors from '../selectors/select';

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

export function selectDeploymentFilter(deploymentName) {
    cy.intercept('GET', api.network.networkGraph).as('networkGraph');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('networkPolicies');
    cy.get(networkGraphSelectors.toolbar.filterSelect).type('Deployment{enter}');
    cy.get(networkGraphSelectors.toolbar.filterSelect).type(`${deploymentName}{enter}{esc}`);
    cy.wait(['@networkGraph', '@networkPolicies']);
}

export function selectNamespaceFilters(...namespaces) {
    cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
    namespaces.forEach((ns) => {
        cy.contains(`${selectSelectors.patternFlySelect.openMenu} span`, ns).click();
    });
    cy.get(networkGraphSelectors.toolbar.namespaceSelect).click();
}

// visit helpers

export function visitNetworkGraphWithNamespaceFilters(...namespaces) {
    cy.intercept('GET', api.clusters.list).as('clusters');
    cy.visit(networkUrl);
    cy.wait('@clusters');

    cy.intercept('GET', api.network.networkGraph).as('networkGraph');
    cy.intercept('GET', api.network.networkPoliciesGraph).as('networkPolicies');
    selectNamespaceFilters(...namespaces);
    cy.wait(['@networkGraph', '@networkPolicies']);
}

export function visitNetworkGraphWithMockedData() {
    cy.intercept('GET', api.clusters.list).as('clusters');
    cy.intercept('GET', api.network.networkGraph, { fixture: 'network/networkGraph.json' }).as(
        'networkGraph'
    );
    cy.intercept('GET', api.network.networkPoliciesGraph, {
        fixture: 'network/networkPolicies.json',
    }).as('networkPolicies');

    cy.visit(networkUrl);
    cy.wait('@clusters');
    selectNamespaceFilters('stackrox');
    cy.wait(['@networkGraph', '@networkPolicies']);
}
