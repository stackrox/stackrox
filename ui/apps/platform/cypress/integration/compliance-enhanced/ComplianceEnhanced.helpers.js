import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance-enhanced';
export const statusDashboardPath = `${basePath}/status`;
export const clusterCompliancePath = `${basePath}/cluster-compliance`;
export const clusterComplianceCoveragePath = `${clusterCompliancePath}/coverage`;
export const clusterComplianceScanConfigsPath = `${clusterCompliancePath}/scan-configs`;

// visit helpers
export const scanConfigsAlias = 'configurations';

// TODO: (vjw, 13 Nov 2023) after the API endpoints for the dashboard are published,
// we will add them to the routeMatcherMap here
const routeMatcherMapForComplianceDashboard = null;

const routeMatcherMapForComplianceScanConfigs = {
    [scanConfigsAlias]: {
        method: 'GET',
        url: '/v2/compliance/scan/configurations*',
    },
};

export function visitComplianceEnhancedFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable(
        'Compliance (2.0)',
        'Compliance Status',
        routeMatcherMapForComplianceDashboard,
        staticResponseMap
    );
}

export function visitComplianceEnhancedDashboard(staticResponseMap) {
    // TODO: (vjw, 1 Nov 2023) add routes matchers to this function, after API is available for Status
    visit(basePath, routeMatcherMapForComplianceDashboard, staticResponseMap);
    cy.get(`h1:contains("Compliance")`);
}

export function visitComplianceEnhancedClusterComplianceFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable('Compliance (2.0)', 'Cluster Compliance', null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`);
}

export function visitComplianceEnhancedClusterCompliance(staticResponseMap) {
    visit(clusterCompliancePath, null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`);
}

export function visitComplianceEnhancedScanConfigs(staticResponseMap) {
    visit(
        clusterComplianceScanConfigsPath,
        routeMatcherMapForComplianceScanConfigs,
        staticResponseMap
    );

    cy.get('a.pf-c-nav__link').contains('Schedules').should('have.class', 'pf-m-current');
}
