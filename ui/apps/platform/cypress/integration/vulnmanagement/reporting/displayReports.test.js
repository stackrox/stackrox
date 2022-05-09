import { selectors as VulnMgmtPageSelectors } from '../../../constants/VulnManagementPage';
import withAuth from '../../../helpers/basicAuth';
import { visitVulnerabilityReportingWithFixture } from '../../../helpers/vulnmanagement/reporting';

describe('Vulnmanagement reports', () => {
    withAuth();

    describe('report configurations', () => {
        it('should show a list of report configurations', () => {
            visitVulnerabilityReportingWithFixture('reports/reportConfigurations.json');

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
