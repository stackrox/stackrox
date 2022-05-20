import * as api from '../../constants/apiEndpoints';
import { headingPlural, url } from '../../constants/VulnManagementPage';

import { visitFromLeftNavExpandable } from '../nav';
import { visit } from '../visit';

const opnamesForDashboard = [
    'policiesCount',
    'cvesCount',
    'getNodes',
    'getImages',
    'topRiskyDeployments',
    'topRiskiestImages',
    'frequentlyViolatedPolicies',
    'recentlyDetectedVulnerabilities',
    'mostCommonVulnerabilities',
    'deploymentsWithMostSeverePolicyViolations',
];

export function visitVulnerabilityManagementDashboardFromLeftNav() {
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });

    visitFromLeftNavExpandable('Vulnerability Management', 'Dashboard');

    cy.wait(opnamesForDashboard.map((opname) => `@${opname}`));
    cy.get('h1:contains("Vulnerability Management")');
}

export function visitVulnerabilityManagementDashboard() {
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });

    visit(url.dashboard);

    cy.wait(opnamesForDashboard.map((opname) => `@${opname}`));
    cy.get('h1:contains("Vulnerability Management")');
}

/*
 * For example, visitEntities('cves')
 * For example, visitEntities('policies', '?s[Policy]=Fixable Severity at least Important')
 */
export function visitVulnerabilityManagementEntities(entitiesKey, search = '') {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.vulnMgmt.graphqlEntities(entitiesKey)).as(entitiesKey);

    visit(`${url.list[entitiesKey]}${search}`);

    cy.wait(['@searchOptions', `@${entitiesKey}`]);
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}
