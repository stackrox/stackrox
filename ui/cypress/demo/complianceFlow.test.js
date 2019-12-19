import { url, selectors as ComplianceSelectors } from '../constants/CompliancePage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

describe('Compliance Flow', () => {
    withAuth();

    it('dashboard loads with scanned data already present', () => {
        cy.visit(url.dashboard);
        cy.get(selectors.page.pageHeader).contains('Compliance');
        cy.get(ComplianceSelectors.scanButton).should('not.have.attr', 'disabled');
        cy.wait(2000);
        cy.get(ComplianceSelectors.emptyMessage).should('not.exist');
    });

    it('clicking on "Passing Standards by Cluster" should take user to NIST standard details with filters: "Cluster: production, Standard: NIST SP 800-190"', () => {
        cy.visit(url.dashboard);
        cy.get(ComplianceSelectors.widget.passingStandardsByCluster.NISTBarLinks)
            .eq(0)
            .click();
        cy.get(selectors.page.pageHeader).contains('NIST');
        cy.get(selectors.search.chips)
            .eq(0)
            .contains('Cluster:');
        cy.get(selectors.search.chips)
            .eq(1)
            .contains('production');
        cy.get(selectors.search.chips)
            .eq(2)
            .contains('Standard:');
        cy.get(selectors.search.chips)
            .eq(3)
            .contains('NIST SP 800-190');
        cy.get(`${selectors.table.rows}:contains("4.1.1") div`)
            .eq(3)
            .contains('100%');
    });

    it('when selecting "NIST 4.1.1", control details pane opens and shows details for NIST 4.1.1 control', () => {
        const controlName = '4.1.1';

        cy.visit(url.list.standards.NIST_800_190);
        cy.get(selectors.table.rows)
            .eq(0)
            .click();
        cy.get(selectors.panel.sidePanel).should('exist');
        cy.get(selectors.table.activeRow).contains(controlName);
        cy.get(selectors.panel.sidePanelHeader).contains(controlName);
        cy.get(ComplianceSelectors.widget.controlDetails.widget).should('exist');
        cy.get(ComplianceSelectors.widget.controlDetails.controlName).contains(controlName);
    });

    it('Compliance Scan button should be disabled while scanning and update data afterwards', () => {
        cy.visit(url.dashboard);
        cy.get(ComplianceSelectors.scanButton).click();
        cy.get(ComplianceSelectors.scanButton).should('have.attr', 'disabled');
    });

    it('clicking Export -> PDF should export to PDF', () => {
        cy.visit(url.dashboard);
        cy.get(ComplianceSelectors.export.exportButton).click();
        cy.get(ComplianceSelectors.export.pdfButton).click();
        cy.get('div:contains("Exporting")').should('exist');

        // TO DO: ROX-3477 test downloaded data
    });

    it('clicking Export -> CSV should export to CSV', () => {
        cy.visit(url.dashboard);
        cy.get(ComplianceSelectors.export.exportButton).click();
        cy.get(ComplianceSelectors.export.csvButton).click();

        // TO DO: ROX-3477 test downloaded data
    });
});
