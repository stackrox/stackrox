import { visitFromHorizontalNav } from '../../helpers/nav';
import { interactAndVisitNetworkGraphWithDeploymentSelected } from '../../helpers/networkGraph';
import { visit } from '../../helpers/visit';

// visit

const basePath = '/main/risk/workloads';

export const deploymentswithprocessinfoAlias = 'deploymentswithprocessinfo';
export const deploymentscountAlias = 'deploymentscount';

// The compound search filter uses a static curated config, so the page no longer issues the
// searchOptions GraphQL query that the legacy SearchFilterInput relied on.
const routeMatcherMap = {
    [deploymentswithprocessinfoAlias]: {
        method: 'GET',
        url: '/v1/deploymentswithprocessinfo*', // wildcard for ?query=
    },
    [deploymentscountAlias]: {
        method: 'GET',
        url: '/v1/deploymentscount*',
    },
};

function getRiskURL(filteredWorkflowView, search = '') {
    if (filteredWorkflowView) {
        return `${basePath}?filteredWorkflowView=${filteredWorkflowView}&${search}`;
    }
    return `${basePath}?${search}`;
}

export function visitRiskDeployments(filteredWorkflowView) {
    visit(getRiskURL(filteredWorkflowView), routeMatcherMap);

    cy.get('h1:contains("Risk")');
}

export function visitRiskDeploymentsWithSearchQuery(filteredWorkflowView, search) {
    const riskURL = getRiskURL(filteredWorkflowView, search);
    visit(riskURL, routeMatcherMap);

    cy.get('h1:contains("Risk")');
}

export function viewFirstRiskDeployment() {
    return cy.get('tbody tr:first td[data-label="Name"]').then(($td) => {
        const deploymentName = $td.text().trim();

        cy.get('tbody tr:first td[data-label="Name"] a').click();
        cy.get(`h1:contains("${deploymentName}")`);

        return cy.wrap(deploymentName);
    });
}

export function viewRiskDeploymentInNetworkGraph(deploymentName) {
    interactAndVisitNetworkGraphWithDeploymentSelected(deploymentName, () => {
        cy.get('a:contains("View Deployment in Network Graph")').click();
    });
}

// interact

/**
 * @param {'User Workloads'|'Platform'|'All Deployments'} viewName
 */
export function selectFilteredWorkflowView(viewName) {
    visitFromHorizontalNav(viewName);
}
