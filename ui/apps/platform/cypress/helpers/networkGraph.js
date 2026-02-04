import * as api from '../constants/apiEndpoints';
import { interactAndWaitForResponses } from './request';

const networkGraphClusterAlias = 'networkgraph/cluster/id';

const routeMatcherMapForClusterInNetworkGraph = {
    [networkGraphClusterAlias]: {
        method: 'GET',
        url: '/v1/networkgraph/cluster/*',
    },
};

export const clustersAlias = 'clusters';
export const networkPoliciesGraphEpochAlias = 'networkpolicies/graph/epoch';
export const searchMetadataOptionsAlias = 'search/metadata/options';
export const namespaceAlias = 'namespaces';

const routeMatcherMapToVisitNetworkGraph = {
    [clustersAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters?permissions=**',
    },
    [searchMetadataOptionsAlias]: {
        method: 'GET',
        url: api.search.optionsCategories('DEPLOYMENTS'),
    },
    [namespaceAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters/*/namespaces?permissions**',
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

/**
 * Visit network graph deployment by interaction from another container.
 * For example, click View Deployment in Network Graph button from Risk.
 *
 * @param {function} interactionCallback
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function interactAndVisitNetworkGraphWithDeploymentSelected(
    deploymentName,
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
    cy.get(`[role="dialog"] h2:contains("${deploymentName}")`);
    cy.get(
        `g[data-kind="graph"] [data-kind="node"] .pf-m-selected text:contains("${deploymentName}")`
    );
}
