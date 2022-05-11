import * as api from '../../constants/apiEndpoints';
import { url } from '../../constants/VulnManagementPage';

import { visitFromLeftNavExpandable } from '../nav';
import { visit } from '../visit';

// visit

export function visitVulnerabilityReportingFromLeftNav() {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('GET', api.report.configurations).as('getReportConfigurations');
    cy.intercept('GET', api.report.configurationsCount).as('getReportConfigurationsCount');

    visitFromLeftNavExpandable('Vulnerability Management', 'Reporting');

    cy.wait(['@searchOptions', '@getReportConfigurations', '@getReportConfigurationsCount']);
    cy.get('h1:contains("Vulnerability reporting")');
}

export function visitVulnerabilityReporting() {
    cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
    cy.intercept('GET', api.report.configurations).as('getReportConfigurations');
    cy.intercept('GET', api.report.configurationsCount).as('getReportConfigurationsCount');

    visit(url.reporting.list);

    cy.wait(['@searchOptions', '@getReportConfigurations', '@getReportConfigurationsCount']);
    cy.get('h1:contains("Vulnerability reporting")');
}

export function visitVulnerabilityReportingWithFixture(fixturePath) {
    cy.fixture(fixturePath).then(({ reportConfigs }) => {
        cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
        cy.intercept('GET', api.report.configurations, {
            body: { reportConfigs },
        }).as('getReportConfigurations');
        cy.intercept('GET', api.report.configurationsCount, {
            body: { count: reportConfigs.length },
        }).as('getReportConfigurationsCount');

        visit(url.reporting.list);

        cy.wait(['@searchOptions', '@getReportConfigurations', '@getReportConfigurationsCount']);
        cy.get('h1:contains("Vulnerability reporting")');
    });
}
