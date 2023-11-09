import { visitFromLeftNavExpandable } from '../../helpers/nav';
import { visit } from '../../helpers/visit';

// path

export const basePath = '/main/compliance-enhanced';
export const statusDashboardPath = `${basePath}/status`;
export const scanConfigsPath = `${basePath}/scan-configs`;

// visit helpers
export const scanConfigsAlias = 'configurations';

const routeMatcherMapToComplianceEnhancedDashboard = null;

const routeMatcherMapToComplianceEnhancedScanConfigs = {
    [scanConfigsAlias]: {
        method: 'GET',
        url: '/v2/compliance/scan/configurations*',
    },
};

export function visitComplianceEnhancedFromLeftNav() {
    visitFromLeftNavExpandable(
        'Compliance (2.0)',
        'Compliance Status',
        routeMatcherMapToComplianceEnhancedDashboard
    );

    cy.location('pathname').should('eq', statusDashboardPath);
}

export function visitComplianceEnhancedDashboard() {
    // TODO: (vjw, 1 Nov 2023) add routes matchers to this function, after API is available for Status
    visit(basePath, routeMatcherMapToComplianceEnhancedDashboard);

    cy.location('pathname').should('eq', statusDashboardPath);
    cy.get(`h1:contains("Compliance")`);
}

export function visitComplianceEnhancedScanConfigsFromLeftNav() {
    visitFromLeftNavExpandable(
        'Compliance (2.0)',
        'Scheduling',
        routeMatcherMapToComplianceEnhancedScanConfigs
    );

    cy.location('pathname').should('eq', scanConfigsPath);
    cy.get(`h1:contains("Scan schedules")`);
}

export function visitComplianceEnhancedScanConfigs() {
    // TODO: (vjw, 1 Nov 2023) add routes matchers to this function, after API is available for Status
    visit(scanConfigsPath, routeMatcherMapToComplianceEnhancedDashboard);

    cy.get(`h1:contains("Scan schedules")`);
}
