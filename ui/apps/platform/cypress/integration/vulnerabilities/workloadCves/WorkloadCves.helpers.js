import { visit } from '../../../helpers/visit';
import { selectors } from './WorkloadCves.selectors';

const basePath = '/main/vulnerabilities/workload-cves/';

export function visitWorkloadCveOverview() {
    visit(basePath);

    cy.get('h1:contains("Workload CVEs")');
    cy.location('pathname').should('eq', basePath);
}

/**
 * Apply default filters to the workload CVE overview page.
 * @param {('Critical' | 'Important' | 'Moderate' | 'Low')[]} severities
 * @param {('Fixable' | 'Not fixable')[]} fixabilities
 */
export function applyDefaultFilters(severities, fixabilities) {
    cy.get('button:contains("Default vulnerability filters")').click();
    severities.forEach((severity) => {
        cy.get(`label:contains("${severity}")`).click();
    });
    fixabilities.forEach((severity) => {
        cy.get(`label:contains("${severity}")`).click();
    });
    cy.get('button:contains("Apply filters")').click();
}

/**
 * Apply local severity filters via the filter toolbar.
 * @param {...('Critical' | 'Important' | 'Moderate' | 'Low')} severities
 */
export function applyLocalSeverityFilters(...severities) {
    cy.get(selectors.severityDropdown).click();
    severities.forEach((severity) => {
        cy.get(selectors.severityMenuItem(severity)).click();
    });
    cy.get(selectors.severityDropdown).click();
}

/**
 * Select a resource filter type from the resource filter dropdown.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace')} entityType
 */
export function selectResourceFilterType(entityType) {
    cy.get(selectors.resourceDropdown).click();
    cy.get(selectors.resourceMenuItem(entityType)).click();
    cy.get(selectors.resourceDropdown).click();
}

/**
 * Type a value into the resource filter typeahead and select the first matching value.
 * @param {('CVE' | 'Image' | 'Deployment' | 'Cluster' | 'Namespace')} entityType
 * @param {string} value
 */
export function typeAndEnterResourceFilterValue(entityType, value) {
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
    cy.get(selectors.resourceValueTypeahead(entityType)).type(value);
    cy.get(selectors.resourceValueMenuItem(entityType, value)).click();
    cy.get(selectors.resourceValueTypeahead(entityType)).click();
}

/**
 * View a specific entity tab for a Workload CVE table
 *
 * @param {('CVE' | 'Image' | 'Deployment')} entityType
 */
export function selectEntityTab(entityType) {
    cy.get(selectors.entityTypeToggleItem(entityType)).click();
}
