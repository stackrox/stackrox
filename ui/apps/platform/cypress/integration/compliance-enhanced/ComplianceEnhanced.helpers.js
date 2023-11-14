import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance-enhanced';
export const statusDashboardPath = `${basePath}/status`;
export const scanConfigsPath = `${basePath}/scan-configs`;

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

export function visitComplianceEnhancedScanConfigsFromLeftNav(staticResponseMap) {
    visitFromLeftNavExpandable(
        'Compliance (2.0)',
        'Scheduling',
        routeMatcherMapForComplianceScanConfigs,
        staticResponseMap
    );

    cy.get(`h1:contains("Scan schedules")`);
}

export function visitComplianceEnhancedScanConfigs(staticResponseMap) {
    visit(scanConfigsPath, routeMatcherMapForComplianceScanConfigs, staticResponseMap);

    cy.get(`h1:contains("Scan schedules")`);
}
