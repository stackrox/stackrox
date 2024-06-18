import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance';
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
    visitFromLeftNavExpandable('Compliance', 'Coverage', null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`); // TODO obsolete but function is not called
}

export function visitComplianceEnhancedSchedulesFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable('Compliance', 'Schedules', null, staticResponseMap);

    cy.get(`h1:contains("Schedules")`);
}

export function visitComplianceEnhancedCoverage(staticResponseMap) {
    visit(complianceEnhancedCoveragePath, null, staticResponseMap);

    cy.get(`h1:contains("Cluster compliance")`); // TODO obsolete but function is not called
}

export function visitComplianceEnhancedScanConfigs(staticResponseMap) {
    visit(
        complianceEnhancedScanConfigsPath,
        routeMatcherMapForComplianceScanConfigs,
        staticResponseMap
    );

    cy.get('a.pf-v5-c-nav__link:contains("Schedules")').should('have.class', 'pf-m-current');
}
