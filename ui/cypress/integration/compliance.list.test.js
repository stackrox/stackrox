import { url, selectors as ComplianceSelectors } from '../constants/CompliancePage';
import selectors from '../selectors/index';
import withAuth from '../helpers/basicAuth';

describe('Compliance list page', () => {
    withAuth();

    it('should filter the table with passing controls', () => {
        cy.visit(url.list.namespaces);
        cy.get(selectors.search.input)
            .type('Compliance State:')
            .type('{enter}');
        cy.get(selectors.search.input)
            .type('Pass')
            .type('{enter}');
        cy.get(selectors.table.rows).should('not.exist');
    });

    it('should filter the table with failing controls', () => {
        cy.visit(url.list.namespaces);
        cy.get(selectors.search.input)
            .type('Compliance State:')
            .type('{enter}');
        cy.get(selectors.search.input)
            .type('Fail')
            .type('{enter}');
        cy.get(selectors.table.rows);
    });

    it('should open/close side panel when clicking on a table row', () => {
        cy.visit(url.list.clusters);
        cy.get(ComplianceSelectors.list.table.firstRowName)
            .invoke('text')
            .then(name => {
                cy.get(ComplianceSelectors.list.table.firstRow).click();
                cy.get(selectors.panel.sidePanel).should('exist');
                cy.get(selectors.panel.sidePanelHeader).contains(name);
                cy.get(ComplianceSelectors.widget.relatedEntities).should('not.exist');
                cy.get(selectors.panel.closeButton).click();
                cy.get(selectors.panel.sidePanel).should('not.exist');
            });
    });

    it('should link to entity page when clicking on side panel header', () => {
        cy.visit(url.list.clusters);
        cy.get(ComplianceSelectors.list.table.firstRowName)
            .invoke('text')
            .then(name => {
                cy.get(ComplianceSelectors.list.table.firstRow).click();
                cy.get(selectors.panel.sidePanelHeader).contains(name);
                cy.get(selectors.panel.sidePanelHeader).click();
                cy.url().should('include', url.entity.cluster);
            });
    });

    it('should be sorted by version in standards list', () => {
        cy.visit(url.list.standards.CIS_Docker_v1_2_0);
        cy.get(ComplianceSelectors.list.table.firstRowName)
            .invoke('text')
            .then(text1 => {
                cy.get(ComplianceSelectors.list.table.secondRowName)
                    .invoke('text')
                    .then(text2 => {
                        const arr1 = text1.split(' ')[0];
                        const controlArr1 = arr1.split('.');
                        const arr2 = text2.split(' ')[0];
                        const controlArr2 = arr2.split('.');
                        expect(controlArr1[0]).to.be.at.most(controlArr2[0]);
                        if (controlArr1[1] && controlArr2[1]) {
                            expect(controlArr1[1]).to.be.at.most(controlArr2[1]);
                        }
                    });
            });
    });

    it('should collapse/open table grouping', () => {
        cy.visit(url.list.standards.PCI_DSS_3_2);
        cy.get(ComplianceSelectors.list.table.firstTableGroup).should('be.visible');
        cy.get(ComplianceSelectors.list.table.firstGroup).click();
        cy.get(ComplianceSelectors.list.table.firstTableGroup).should('not.be.visible');
    });

    it('should collapse/open table banner', () => {
        cy.visit(url.list.clusters);
        cy.get(ComplianceSelectors.list.banner.content).should('be.visible');
        cy.get(ComplianceSelectors.list.banner.collapseButton).click();
        cy.get(ComplianceSelectors.list.banner.content).should('be.not.visible');
    });

    it('should show the proper percentage value in the gauge in the Standards List page', () => {
        cy.visit(url.list.standards.CIS_Docker_v1_2_0);
        cy.get(ComplianceSelectors.widget.controlsInCompliance.centerLabel)
            .invoke('text')
            .then(labelPercentage => {
                cy.get(ComplianceSelectors.widget.controlsInCompliance.passingControls)
                    .invoke('text')
                    .then(passingControls => {
                        cy.get(ComplianceSelectors.widget.controlsInCompliance.failingControls)
                            .invoke('text')
                            .then(failingControls => {
                                const percentagePassing = Math.round(
                                    (parseInt(passingControls, 10) /
                                        (parseInt(passingControls, 10) +
                                            parseInt(failingControls, 10))) *
                                        100
                                );
                                expect(percentagePassing).to.be.equal(
                                    parseInt(labelPercentage, 10)
                                );
                            });
                    });
            });
    });

    it('should go to the specific control when User clicks an item from the "Controls Most Failed" widget', () => {
        cy.visit(url.list.standards.CIS_Docker_v1_2_0);
        cy.get(ComplianceSelectors.widget.controlsMostFailed.listItems, { timeout: 10000 })
            .eq(0)
            .invoke('text')
            .then(text => {
                const controlName = text.split(':')[0];
                cy.get(ComplianceSelectors.widget.controlsMostFailed.listItems)
                    .eq(0)
                    .click();
                cy.get(ComplianceSelectors.widget.controlDetails.controlName)
                    .invoke('text')
                    .should('eq', controlName);
            });
    });
});
