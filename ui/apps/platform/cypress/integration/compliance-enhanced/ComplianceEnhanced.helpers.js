import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance-enhanced';
export const statusDashboardPath = `${basePath}/status`;
export const complianceEnhancedCoveragePath = `${basePath}/coverage`;
export const complianceEnhancedScanConfigsPath = `${basePath}/schedules`;

// visit helpers
export const scanConfigsAlias = 'configurations';

const routeMatcherMapForComplianceScanConfigs = {
    [scanConfigsAlias]: {
        method: 'GET',
        url: '/v2/compliance/scan/configurations*',
    },
};

export function visitComplianceEnhancedCoverageFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable('Compliance (2.0)', 'Coverage', null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`);
}

export function visitComplianceEnhancedSchedulesFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable('Compliance (2.0)', 'Schedules', null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`);
}

export function visitComplianceEnhancedCoverage(staticResponseMap) {
    visit(complianceEnhancedCoveragePath, null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`);
}

export function visitComplianceEnhancedScanConfigs(staticResponseMap) {
    visit(
        complianceEnhancedScanConfigsPath,
        routeMatcherMapForComplianceScanConfigs,
        staticResponseMap
    );

    cy.get('a.pf-v5-c-nav__link').contains('Schedules').should('have.class', 'pf-m-current');
}
