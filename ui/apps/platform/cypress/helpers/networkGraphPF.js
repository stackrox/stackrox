import * as api from '../constants/apiEndpoints';
import { selectors as networkGraphSelectors } from '../constants/NetworkPage';
import { interactAndWaitForResponses } from './request';
import { visit } from './visit';
import selectSelectors from '../selectors/select';
import navSelectors from '../selectors/navigation';
import { visitMainDashboard } from './main';

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

const clusterNamespacesTarget =
    '/v1/sac/clusters/*/namespaces?permissions=NetworkGraph&permissions=Deployment';

export function selectCluster() {
    cy.intercept('GET', clusterNamespacesTarget);

    // no longer necessary to await getting NS, because in one-cluster environments, we now pre-select the cluster
    interactAndWaitForResponses(() => {
        cy.get(networkGraphSelectors.selector.clusterSelect).click();
        cy.get(`${selectSelectors.patternFlySelect.openMenu} span:first`).click();
    });
}

// Additional calls in a test can select additional namespaces.

export function selectNamespace(namespace) {
    interactAndWaitForResponses(() => {
        cy.get(networkGraphSelectors.selector.namespaceSelect).click();
        // Exact match to distinguish stackrox from stackrox-operator namespaces.
        cy.get(
            `${selectSelectors.patternFlySelect.openMenu} .pf-c-menu__list-item [data-testid="namespace-name"]`
        )
            .contains(new RegExp(`^${namespace}$`))
            .click();
        cy.get(networkGraphSelectors.selector.namespaceSelect).click();
    }, routeMatcherMapForClusterInNetworkGraph);
}

export function selectDeployment(deployment) {
    interactAndWaitForResponses(() => {
        cy.get(networkGraphSelectors.selector.deploymentSelect).click();
        cy.get(
            `${selectSelectors.patternFlySelect.openMenu} .pf-c-menu__list-item [data-testid="deployment-name"]`
        )
            .contains(new RegExp(`^${deployment}$`))
            .click();
        cy.get(networkGraphSelectors.selector.deploymentSelect).click();
    }, routeMatcherMapForClusterInNetworkGraph);
}

// visit helpers

export const notifiersAlias = 'notifiers';
export const clustersAlias = 'clusters';
export const networkPoliciesGraphEpochAlias = 'networkpolicies/graph/epoch';
export const searchMetadataOptionsAlias = 'search/metadata/options';

const routeMatcherMapToVisitNetworkGraph = {
    [clustersAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters?permissions=NetworkGraph&permissions=Deployment',
    },
    [networkPoliciesGraphEpochAlias]: {
        method: 'GET',
        url: `${api.network.epoch}?clusterId=*`, // either id or null if no cluster selected
    },
};

export const basePath = '/main/network-graph';

// TODO: replace this custom implementation with the version from
//    import { visitFromLeftNav } from './nav';
// after the old network graph goes away in the left nav
function visitFromLeftNav(itemText, routeMatcherMap, staticResponseMap) {
    visitMainDashboard();

    interactAndWaitForResponses(
        () => {
            cy.get(`${navSelectors.navLinks}:contains("${itemText}")`).first().click();
        },
        routeMatcherMap,
        staticResponseMap
    );
}

export function visitNetworkGraphFromLeftNav() {
    visitFromLeftNav('Network Graph', routeMatcherMapToVisitNetworkGraph);

    cy.location('pathname').should('eq', basePath);
}

export function visitNetworkGraph(staticResponseMap) {
    visit(basePath, routeMatcherMapToVisitNetworkGraph, staticResponseMap);
}

export function checkNetworkGraphEmptyState() {
    cy.get(
        '.pf-c-empty-state__content:contains("Select a cluster and at least one namespace to render active deployment traffic on the graph")'
    );
}
