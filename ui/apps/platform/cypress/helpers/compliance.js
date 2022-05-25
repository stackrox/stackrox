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

const standardNames = [
    'CIS Docker v1.2.0',
    'CIS Kubernetes v1.5',
    'HIPAA 164',
    'NIST SP 800-190',
    'NIST SP 800-53',
    'PCI DSS 3.2.1',
];

export function visitComplianceDashboard() {
    opnamesForDashboard.forEach((opname) => {
        cy.intercept('POST', api.graphql(opname)).as(opname);
    });
    // Intercept requests for compliance standards, which have same opname but different value in payload.
    cy.intercept('POST', api.graphql('complianceStandards'), (req) => {
        const { where } = req.body.variables;
        const alias = standardNames.find((standardName) => where === `Standard:${standardName}`);
        if (typeof alias === 'string') {
            req.alias = alias;
        }
    });

    visit(url.dashboard);

    cy.wait(opnamesForDashboard.map((opname) => `@${opname}`));
    cy.wait(standardNames.map((standardName) => `@${standardName}`));
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

    cy.get(selectors.scanButton).should('not.have.attr', 'disabled');
    cy.get(selectors.scanButton).click();
    cy.get(selectors.scanButton).should('have.attr', 'disabled');

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
 */
export function visitComplianceStandard(standardName) {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('POST', api.graphql('getComplianceStandards')).as('getComplianceStandards');
    cy.intercept('POST', api.graphql('controls')).as('controls');

    visit(`${url.controls}?s[standard]=${standardName}`);

    cy.wait(['@searchOptions', '@getComplianceStandards', '@controls']);
    cy.get(`h1:contains("${standardName}")`);
}
