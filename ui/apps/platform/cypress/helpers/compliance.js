import * as api from '../constants/apiEndpoints';
import { headingPlural, selectors, url } from '../constants/CompliancePage';

import { interceptRequests, waitForResponses } from './request';
import { visit } from './visit';

const routeMatcherMap = {};
[
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
].forEach((opname) => {
    routeMatcherMap[opname] = {
        method: 'POST',
        url: api.graphql(opname),
    };
});

const requestConfig = { routeMatcherMap };

export function visitComplianceDashboard() {
    visit(url.dashboard, requestConfig);

    cy.get('h1:contains("Compliance")');
}

/*
 * Assume location is compliance dashboard.
 */
export function scanCompliance() {
    cy.intercept('POST', api.graphql('triggerScan')).as('triggerScan');
    interceptRequests(requestConfig);

    cy.get(selectors.scanButton).should('not.have.attr', 'disabled');
    cy.get(selectors.scanButton).click();
    cy.get(selectors.scanButton).should('have.attr', 'disabled');

    cy.wait('@triggerScan');
    waitForResponses(requestConfig);

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

/*
 * For example, visitComplianceEntities('clusters')
 */
export function visitComplianceEntities(entitiesKey) {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.compliance.graphqlEntities(entitiesKey)).as(entitiesKey);

    visit(url.entities[entitiesKey]);

    cy.wait(['@searchOptions', `@${entitiesKey}`]);
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}

/*
 * For example, visitComplianceStandard('CIS Docker v1.2.0')
 */
export function visitComplianceStandard(standardName) {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.graphql('getComplianceStandards')).as('getComplianceStandards');
    cy.intercept('POST', api.graphql('controls')).as('controls');

    visit(`${url.controls}?s[standard]=${standardName}`);

    cy.wait(['@searchOptions', '@getComplianceStandards', '@controls']);
    cy.get(`h1:contains("${standardName}")`);
}
