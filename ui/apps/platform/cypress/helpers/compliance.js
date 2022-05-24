import * as api from '../constants/apiEndpoints';
import { headingPlural, selectors, url } from '../constants/CompliancePage';

import { visit } from './visit';

const opnamesForDashboard = [
    'clustersCount',
    'namespacesCount',
    'nodesCount',
    'deploymentsCount',
    'runStatuses',
    'getComplianceStandards',
];

export function visitComplianceDashboard() {
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });

    visit(url.dashboard);

    cy.wait(opnamesForDashboard.map((opname) => `@${opname}`));
    cy.get('h1:contains("Compliance")');
}

/*
 * Assume location is compliance dashboard.
 */
export function scanCompliance() {
    cy.intercept('POST', api.graphql('triggerScan')).as('triggerScan');
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });

    cy.get(selectors.scanButton).click().should('have.attr', 'disabled');

    cy.wait('@triggerScan'); // request occurs immediately
    cy.wait(
        opnamesForDashboard.map((opname) => `@${opname}`),
        {
            requestTimeout: 30000, // increase from default 5 seconds until requests occur
        }
    );
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
 * For example, visitComplianceStandard('CIS Docker v1.2.0', 'TODO')
 */
export function visitComplianceStandard(standardName, searchSuffix = '') {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.graphql('getComplianceStandards')).as('getComplianceStandards');
    cy.intercept('POST', api.graphql('controls')).as('controls');

    visit(`${url.controls}?s[standard]=${standardName}${searchSuffix}`);

    cy.wait(['@searchOptions', '@getComplianceStandards', '@controls']);
    cy.get(`h1:contains("${standardName}")`);
}
