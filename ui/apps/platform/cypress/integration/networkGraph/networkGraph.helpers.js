import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';
import { networkGraphSelectors } from './networkGraph.selectors';

const networkGraphClusterAlias = 'networkgraph/cluster/id';

const routeMatcherMapForClusterInNetworkGraph = {
    [networkGraphClusterAlias]: {
        method: 'GET',
        url: '/v1/networkgraph/cluster/*',
    },
};

// select

const navSelector = 'nav[aria-label="Breadcrumb"]';
const clusterSelect = `${navSelector} [aria-label="Select a cluster"]`;
const namespaceSelect = `${navSelector} [aria-label="Select namespaces"]`;
const deploymentSelect = `${navSelector} [aria-label="Select deployments"]`;
const patternFlyOpenMenu = '.pf-v5-c-menu';

const clusterNamespacesTarget = '/v1/sac/clusters/*/namespaces?permissions=*';

export function selectCluster() {
    cy.intercept('GET', clusterNamespacesTarget);

    // no longer necessary to await getting NS, because in one-cluster environments, we now pre-select the cluster
    interactAndWaitForResponses(() => {
        cy.get(clusterSelect).click();
        cy.get(`${patternFlyOpenMenu} .pf-v5-c-menu__item:first-child`).click();
    });
}

// Additional calls in a test can select additional namespaces.

export function selectNamespace(namespace) {
    interactAndWaitForResponses(() => {
        cy.get(namespaceSelect).click();
        // Exact match to distinguish stackrox from stackrox-operator namespaces.
        cy.get(`${patternFlyOpenMenu} .pf-v5-c-menu__list-item [data-testid="namespace-name"]`)
            .contains(new RegExp(`^${namespace}$`))
            .click();
        cy.get(namespaceSelect).click();
    }, routeMatcherMapForClusterInNetworkGraph);
}

export function selectDeployment(deployment) {
    interactAndWaitForResponses(() => {
        cy.get(deploymentSelect).click();
        cy.get(`${patternFlyOpenMenu} .pf-v5-c-menu__list-item [data-testid="deployment-name"]`)
            .contains(new RegExp(`^${deployment}$`))
            .click();
        cy.get(deploymentSelect).click();
    }, routeMatcherMapForClusterInNetworkGraph);
}

export function selectFilter(filterKey, filterValue) {
    cy.get('.react-select__value-container').click();
    cy.get(`.react-select__menu-list .react-select__option:contains("${filterKey}")`).click();
    cy.focused().type(filterValue);
    cy.get(`.react-select__menu-list .react-select__option:contains("${filterValue}")`)
        .first()
        .click();
    cy.get('.react-select__value-container').click();
}

// visit helpers

export const notifiersAlias = 'notifiers';
export const clustersAlias = 'clusters';
// Removed the following because request has 30 second delay from polling interval:
// export const networkPoliciesGraphEpochAlias = 'networkpolicies/graph/epoch';
export const searchMetadataOptionsAlias = 'search/metadata/options';

const routeMatcherMapToVisitNetworkGraph = {
    [clustersAlias]: {
        method: 'GET',
        url: '/v1/sac/clusters?permissions=*',
    },
};

export const basePath = '/main/network-graph';

export function visitNetworkGraphFromLeftNav() {
    visitFromLeftNavExpandable('Network', 'Network Graph', routeMatcherMapToVisitNetworkGraph);

    cy.location('pathname').should('eq', basePath);
}

export function visitNetworkGraph(staticResponseMap) {
    visit(basePath, routeMatcherMapToVisitNetworkGraph, staticResponseMap);
}

export function checkNetworkGraphEmptyState() {
    cy.get(
        '.pf-v5-c-empty-state__content:contains("Select a cluster and at least one namespace to render active deployment traffic on the graph")'
    );
}

export function updateAndCloseCidrModal() {
    cy.clock();
    cy.get(networkGraphSelectors.updateCidrBlocksButton).click();
    cy.get(
        networkGraphSelectors.cidrModalAlertWithMessage(
            'CIDR blocks have been successfully configured'
        )
    );
    // Once the above alert is shown, the modal automatically closes after 2000 ms. This
    // advances the clock to save time during test runs. (Otherwise every save would add 2 seconds
    // to our test job.)
    cy.tick(2000);
    cy.get(networkGraphSelectors.manageCidrBlocksModal).should('not.exist');
}
