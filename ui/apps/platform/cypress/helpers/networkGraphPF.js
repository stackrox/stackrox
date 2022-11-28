import * as api from '../constants/apiEndpoints';
import { selectors as networkGraphSelectors } from '../constants/NetworkPage';
import { visitFromLeftNav } from './nav';
import { interactAndWaitForResponses } from './request';
import { visit } from './visit';
import selectSelectors from '../selectors/select';

export const networkBaselineStatusAlias = 'networkbaseline/id/status';

// search filters

const networkGraphClusterAlias = 'networkgraph/cluster/id';
const networkPoliciesClusterAlias = 'networkpolicies/cluster/id';

const routeMatcherMapForClusterInNetworkGraph = {
    [networkGraphClusterAlias]: {
        method: 'GET',
        url: api.network.networkGraph,
    },
    [networkPoliciesClusterAlias]: {
        method: 'GET',
        url: api.network.networkPoliciesGraph,
    },
};

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
        routeMatcherMapForClusterInNetworkGraph,
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
export const getClusterNamespaceNamesOpname = 'getClusterNamespaceNames';

const routeMatcherMapToVisitNetworkGraph = {
    // [notifiersAlias]: {
    //     method: 'GET',
    //     url: api.integrations.notifiers,
    // },
    [clustersAlias]: {
        method: 'GET',
        url: api.clusters.list,
    },
    [networkPoliciesGraphEpochAlias]: {
        method: 'GET',
        url: `${api.network.epoch}?clusterId=*`, // either id or null if no cluster selected
    },
    // [searchMetadataOptionsAlias]: {
    //     method: 'GET',
    //     url: api.search.optionsCategories('DEPLOYMENTS'),
    // },
};

export const deploymentAlias = 'deployments/id';

const routeMatcherMapToVisitNetworkGraphWithDeploymentSelected = {
    ...routeMatcherMapToVisitNetworkGraph,
    [deploymentAlias]: {
        method: 'GET',
        url: '/v1/deployments/*',
    },
    ...routeMatcherMapForClusterInNetworkGraph,
};

export const basePath = '/main/network-graph';

/*
 * Reach clusters by interaction from another container.
 * For example, click View Deployment in Network Graph button from Risk.
 */
export function reachNetworkGraphWithDeploymentSelected(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapToVisitNetworkGraphWithDeploymentSelected,
        staticResponseMap
    );

    cy.location('pathname').should('contain', basePath); // contain because pathname might have id
    cy.get(networkGraphSelectors.networkGraphHeading);
}

export function visitNetworkGraphFromLeftNav() {
    visitFromLeftNav('PatternFly Network Graph', routeMatcherMapToVisitNetworkGraph);

    cy.location('pathname').should('eq', basePath);
    cy.get(networkGraphSelectors.networkGraphHeading);
}

export function visitNetworkGraph(staticResponseMap) {
    visit(basePath, routeMatcherMapToVisitNetworkGraph, staticResponseMap);

    cy.get(networkGraphSelectors.networkGraphHeading);
}
