import * as api from '../constants/apiEndpoints';
import { interactAndWaitForResponses } from './request';

const networkGraphClusterAlias = 'networkgraph/cluster/id';
const networkPoliciesClusterAlias = 'networkpolicies/cluster/id';

const routeMatcherMapForClusterInNetworkGraph = {
    [networkGraphClusterAlias]: {
        method: 'GET',
        url: '/v1/networkgraph/cluster/*',
    },
    [networkPoliciesClusterAlias]: {
        method: 'GET',
        url: '/v1/networkpolicies/cluster/*',
    },
};

export const notifiersAlias = 'notifiers';
export const clustersAlias = 'clusters';
export const networkPoliciesGraphEpochAlias = 'networkpolicies/graph/epoch';
export const searchMetadataOptionsAlias = 'search/metadata/options';
export const namespaceAlias = 'namespaces';

const routeMatcherMapToVisitNetworkGraph = {
    [notifiersAlias]: {
        method: 'GET',
        url: '/v1/notifiers',
    },
    [clustersAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters?permissions=NetworkGraph&permissions=Deployment',
    },
    [networkPoliciesGraphEpochAlias]: {
        method: 'GET',
        url: '/v1/networkpolicies/graph/epoch?clusterId=*', // either id or null if no cluster selected
    },
    [searchMetadataOptionsAlias]: {
        method: 'GET',
        url: api.search.optionsCategories('DEPLOYMENTS'),
    },
    [namespaceAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters/*/namespaces?permissions=NetworkGraph&permissions=Deployment',
    },
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

export const basePath = '/main/network';

const title = 'Network Graph';

/**
 * Visit network graph deployment by interaction from another container.
 * For example, click View Deployment in Network Graph button from Risk.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndVisitNetworkGraphWithDeploymentSelected(
    interactionCallback,
    staticResponseMap
) {
    interactAndWaitForResponses(
        () => {
            interactionCallback();
        },
        routeMatcherMapToVisitNetworkGraphWithDeploymentSelected,
        staticResponseMap
    );

    cy.location('pathname').should('contain', basePath); // contain because pathname has id
    cy.get(`h1:contains("${title}")`);
}
