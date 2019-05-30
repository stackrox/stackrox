import { url, selectors } from './constants/CompliancePage';
import withAuth from './helpers/basicAuth';

describe('Compliance list page', () => {
    withAuth();

    it('should open/close side panel when clicking on a table row', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then(name => {
                cy.get(selectors.list.table.firstRow).click();
                cy.get(selectors.list.panels)
                    .its('length')
                    .should('eq', 2);
                cy.get(selectors.list.sidePanelHeader).contains(name);
                cy.get(selectors.widget.relatedEntities).should('not.exist');
                cy.get(selectors.list.sidePanelCloseBtn).click();
                cy.get(selectors.list.panels)
                    .its('length')
                    .should('eq', 1);
            });
    });

    it('should link to entity page when clicking on side panel header', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.table.firstRow).click();
        cy.get(selectors.list.sidePanelHeader).click();
        cy.url().should('include', url.list.clusters);
    });

    it('should be sorted by version in standards list', () => {
        cy.visit(url.list.standards.CIS_Docker_v1_1_0);
        cy.get(selectors.list.table.firstRowName)
            .invoke('text')
            .then(text1 => {
                cy.get(selectors.list.table.secondRowName)
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
        cy.get(selectors.list.table.firstTableGroup).should('be.visible');
        cy.get(selectors.list.table.firstGroup).click();
        cy.get(selectors.list.table.firstTableGroup).should('not.be.visible');
    });

    it('should collapse/open table banner', () => {
        cy.visit(url.list.clusters);
        cy.get(selectors.list.banner.content).should('be.visible');
        cy.get(selectors.list.banner.collapseButton).click();
        cy.get(selectors.list.banner.content).should('be.not.visible');
    });

    it('should show the proper percentage value in the gauge in the Standards List page', () => {
        cy.visit(url.list.standards.CIS_Docker_v1_1_0);
        cy.get(selectors.widget.controlsInCompliance.centerLabel)
            .invoke('text')
            .then(labelPercentage => {
                cy.get(selectors.widget.controlsInCompliance.passingControls)
                    .invoke('text')
                    .then(passingControls => {
                        cy.get(selectors.widget.controlsInCompliance.failingControls)
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
        cy.visit(url.list.standards.CIS_Docker_v1_1_0);
        cy.get(selectors.widget.controlsMostFailed.listItems, { timeout: 10000 })
            .eq(0)
            .invoke('text')
            .then(text => {
                const controlName = text.split(':')[0];
                cy.get(selectors.widget.controlsMostFailed.listItems)
                    .eq(0)
                    .click();
                cy.get(selectors.widget.controlDetails.controlname)
                    .invoke('text')
                    .should('eq', controlName);
            });
    });
});
