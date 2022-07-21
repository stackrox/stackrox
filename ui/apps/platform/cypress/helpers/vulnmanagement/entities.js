import * as api from '../../constants/apiEndpoints';
import { headingPlural, url } from '../../constants/VulnManagementPage';
import { hasFeatureFlag } from '../features';

import { visitFromLeftNavExpandable } from '../nav';
import { visit } from '../visit';

let opnamesForDashboard = [
    'policiesCount',
    'cvesCount',
    'getNodes',
    'getImages',
    'topRiskyDeployments',
    'topRiskiestImagesOld',
    'topRiskiestImageVulns',
    'frequentlyViolatedPolicies',
    'recentlyDetectedVulnerabilities',
    'recentlyDetectedImageVulnerabilities',
    'mostCommonVulnerabilities',
    'mostCommonImageVulnerabilities',
    'deploymentsWithMostSeverePolicyViolations',
    'clustersWithMostOrchestratorIstioVulnerabilities',
];

if (hasFeatureFlag('ROX_FRONTEND_VM_UPDATES')) {
    opnamesForDashboard = opnamesForDashboard.filter(
        (opname) =>
            opname !== 'recentlyDetectedVulnerabilities' &&
            opname !== 'topRiskiestImagesOld' &&
            opname !== 'mostCommonVulnerabilities'
    );
} else {
    opnamesForDashboard = opnamesForDashboard.filter(
        (opname) =>
            opname !== 'recentlyDetectedImageVulnerabilities' &&
            opname !== 'topRiskiestImageVuln' &&
            opname !== 'mostCommonImageVulnerabilities'
    );
}

export function visitVulnerabilityManagementDashboardFromLeftNav() {
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });

    visitFromLeftNavExpandable('Vulnerability Management', 'Dashboard');
    cy.log(`${opnamesForDashboard}`);

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
 * For example, visitVulnerabilityManagementEntities('cves')
 * For example, visitVulnerabilityManagementEntities('policies', '?s[Policy]=Fixable Severity at least Important')
 */
export function visitVulnerabilityManagementEntities(entitiesKey, search = '') {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.vulnMgmt.graphqlEntities(entitiesKey)).as(entitiesKey);

    visit(`${url.list[entitiesKey]}${search}`);

    cy.wait(['@searchOptions', `@${entitiesKey}`]);
    cy.get(`h1:contains("${headingPlural[entitiesKey]}")`);
}
