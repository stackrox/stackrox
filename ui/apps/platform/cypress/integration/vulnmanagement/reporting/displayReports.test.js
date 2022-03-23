import * as api from '../../../constants/apiEndpoints';
import { url, selectors as VulnMgmtPageSelectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';

describe('Vulnmanagement reports', () => {
    withAuth();

    describe('report configurations', () => {
        beforeEach(() => {
            cy.intercept('GET', api.report.configurations, {
                fixture: 'reports/reportConfigurations.json',
            }).as('getReportConfigurations');
            cy.intercept('GET', api.report.configurationsCount, { count: 2 }).as(
                'getReportConfigurationsCount'
            );
            cy.intercept('POST', api.graphql('searchOptions')).as('searchOptions');
        });

        it('should show a list of report configurations', () => {
            cy.visit(`${url.reporting.list}`);

            cy.wait('@getReportConfigurations');
            cy.wait('@getReportConfigurationsCount');
            cy.wait('@searchOptions');

            // Hard-coded wait is to ameliorate a tenacious flake in CI that has resisted all more gentle solutions
            cy.wait(1000);

            // page title
            cy.get('h1:contains("Vulnerability reporting")');

            // column headings
            cy.get(VulnMgmtPageSelectors.reportSection.table.column.name).should('be.visible');
            cy.get(VulnMgmtPageSelectors.reportSection.table.column.description).should(
                'be.visible'
            );
            cy.get(VulnMgmtPageSelectors.reportSection.table.column.cveFixabilityType).should(
                'be.visible'
            );
            cy.get(VulnMgmtPageSelectors.reportSection.table.column.cveSeverities).should(
                'be.visible'
            );
            cy.get(VulnMgmtPageSelectors.reportSection.table.column.lastRun).should('be.visible');

            // row content
            // name
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(1) td:nth-child(2)`
            ).should('contain', 'Failing report');
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(2) td:nth-child(2)`
            ).should('contain', 'Successful report');

            // fixability
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(1) td:nth-child(4)`
            ).should('contain', 'Fixable, Unfixable');
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(2) td:nth-child(4)`
            ).should('contain', 'Fixable');

            // severities
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(1) td:nth-child(5)`
            ).should('contain', 'CriticalImportantMediumLow');
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(2) td:nth-child(5)`
            ).should('contain', 'Critical');

            // last run
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(1) td:nth-child(6)`
            ).should('contain', 'Error');
            cy.get(
                `${VulnMgmtPageSelectors.reportSection.table.rows}:nth-child(2) td:nth-child(6)`
            ).should('contain', '2022');
        });
    });
});
