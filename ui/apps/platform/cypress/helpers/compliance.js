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
    'getAggregatedResults', // 4 requests
    'getComplianceStandards',
    'complianceStandards', // 6 requests
].forEach((opname) => {
    routeMatcherMap[opname] = {
        method: 'POST',
        url: api.graphql(opname),
    };
});

/*
 * getAggregatedResults opname has 4 requests with 2 duplicates for CLUSTER.
 * Each entity alias has a predicate to match the payload of the corresponding request.
 */
const getAggregatedResults = {};
['CLUSTER', 'NAMESPACE', 'NODE'].forEach((entity) => {
    getAggregatedResults[entity] = (req) => {
        const { groupBy, unit } = req.body.variables;
        // "variables": { "groupBy": ["STANDARD", "CLUSTER"], "unit": "CHECK" }
        return (
            Array.isArray(groupBy) &&
            groupBy[0] === 'STANDARD' &&
            groupBy[1] === entity &&
            unit === 'CHECK'
        );
    };
});

/*
 * complianceStandards opname has 6 requests.
 * Each standard name alias has a predicate to match the payload of the corresponding request.
 */
const complianceStandards = {};
[
    'CIS Docker v1.2.0',
    'CIS Kubernetes v1.5',
    'HIPAA 164',
    'NIST SP 800-190',
    'NIST SP 800-53',
    'PCI DSS 3.2.1',
].forEach((standardName) => {
    complianceStandards[standardName] = (req) => {
        const { where } = req.body.variables;
        return where === `Standard:${standardName}`;
    };
});

const opnameAliasesMap = {
    getAggregatedResults,
    complianceStandards,
};

const waitOptions = {
    requestTimeout: 10000, // because so many requests
    responseTimeout: 20000, // for 6 complianceStandards responses
};

const requestConfig = { routeMatcherMap, opnameAliasesMap, waitOptions };

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

    cy.get(selectors.scanButton).click().should('not.have.attr', 'disabled');
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
