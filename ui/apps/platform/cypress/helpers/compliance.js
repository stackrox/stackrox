import { headingPlural, selectors, url } from '../constants/CompliancePage';

import { getRouteMatcherMapForGraphQL, interactAndWaitForResponses } from './request';
import { visitAndAssertBeforeResponses } from './visit';

const routeMatcherMapForComplianceDashboard = getRouteMatcherMapForGraphQL([
    'clustersCount',
    'namespacesCount',
    'nodesCount',
    'deploymentsCount',
    'runStatuses',
    'getAggregatedResultsAcrossEntity_CLUSTER',
    'getAggregatedResultsByEntity_CLUSTER',
    'getAggregatedResultsAcrossEntity_NAMESPACE',
    'getAggregatedResultsAcrossEntity_NODE',
    'getComplianceStandards',
    'complianceStandards_CIS_Docker_v1_2_0',
    'complianceStandards_CIS_Kubernetes_v1_5',
    'complianceStandards_HIPAA_164',
    'complianceStandards_NIST_800_190',
    'complianceStandards_NIST_SP_800_53_Rev_4',
    'complianceStandards_PCI_DSS_3_2',
]);

const containerTitle = 'Compliance';

export function visitComplianceDashboard() {
    visitAndAssertBeforeResponses(
        url.dashboard,
        () => {
            cy.get(`h1:contains("${containerTitle}")`);
        },
        routeMatcherMapForComplianceDashboard
    );
}

/*
 * Assume location is compliance dashboard.
 */
export function scanCompliance() {
    const routeMatcherMapForTriggerScan = getRouteMatcherMapForGraphQL(['triggerScan']);
    const routeMatcherMap = {
        ...routeMatcherMapForTriggerScan,
        ...routeMatcherMapForComplianceDashboard,
    };

    cy.get(selectors.scanButton).should('not.have.attr', 'disabled');

    interactAndWaitForResponses(() => {
        cy.get(selectors.scanButton).click();
        cy.get(selectors.scanButton).should('have.attr', 'disabled');
    }, routeMatcherMap);

    cy.get(selectors.scanButton).should('not.have.attr', 'disabled');
}

/*
 * Call directly in first test (of a test file for another container) which assumes that scan results are available.
 * Although compliance test file bends the guideline that tests should not depend on side effects within the same test file,
 * configmanagement test files break the guideline if they assume that compliance test file has already run.
 * Cypress 10 apparently no longer runs tests files in a determinate order.
 */
export function triggerScan() {
    visitComplianceDashboard();
    scanCompliance();
}

const opnameForEntities = {
    clusters: 'clustersList', // just clusters would be even better, and so on
    deployments: 'deploymentsList',
    namespaces: 'namespaceList', // singular: too bad, so sad
    nodes: 'nodesList',
};

/*
 * For example, visitComplianceEntities('clusters')
 */
export function visitComplianceEntities(entitiesKey) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        opnameForEntities[entitiesKey],
    ]);

    visitAndAssertBeforeResponses(
        url.entities[entitiesKey],
        () => {
            cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
        },
        routeMatcherMap
    );
}

/*
 * For example, visitComplianceStandard('CIS Docker v1.2.0')
 */
export function visitComplianceStandard(standardName) {
    const routeMatcherMap = getRouteMatcherMapForGraphQL([
        'searchOptions',
        'getComplianceStandards',
        'controls',
    ]);

    visitAndAssertBeforeResponses(
        `${url.controls}?s[standard]=${standardName}`,
        () => {
            cy.get(`h1:contains("${standardName}")`);
        },
        routeMatcherMap
    );
}
