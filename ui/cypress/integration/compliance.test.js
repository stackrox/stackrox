import { compliance as complianceAPIEndpoints } from './constants/apiEndpoints';
import { url, selectors } from './constants/CompliancePage';
import withAuth from './helpers/basicAuth';

describe('Compliance page', () => {
    withAuth();

    beforeEach(() => {
        cy.server();
        cy.route({
            method: 'GET',
            url: complianceAPIEndpoints.export.csv,
            status: 200,
            delay: 200,
            response: {}
        }).as('downloadCSV');
        cy.visit(url.dashboard);
    });

    it('should scan for compliance data from the Dashboard page', () => {
        cy.get(selectors.scanButton).click();
        cy.wait(5000);
    });

    it('should show the same amount of clusters between the Dashboard and List Page', () => {
        cy.get(selectors.dashboard.tileLinks.cluster.value)
            .invoke('text')
            .then(text => {
                const numClusters = Number(text);
                cy.visit(url.list.clusters);
                cy.get(selectors.list.tableRows)
                    .its('length')
                    .should('eq', numClusters);
            });
    });

    it('should show the same amount of namespaces between the Dashboard and List Page', () => {
        cy.get(selectors.dashboard.tileLinks.namespace.value)
            .invoke('text')
            .then(text => {
                const numNamespaces = Number(text);
                cy.visit(url.list.namespaces);
                cy.get(selectors.list.tableRows)
                    .its('length')
                    .should('eq', numNamespaces);
            });
    });

    it('should show the same amount of nodes between the Dashboard and List Page', () => {
        cy.get(selectors.dashboard.tileLinks.node.value)
            .invoke('text')
            .then(text => {
                const numNodes = Number(text);
                cy.visit(url.list.nodes);
                cy.get(selectors.list.tableRows)
                    .its('length')
                    .should('eq', numNodes);
            });
    });

    it('should group by clusters when User clicks the link in "Passing Standards Across Clusters"', () => {
        cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks)
            .eq(0)
            .click();
        cy.url().should('contain', '?groupBy=CLUSTER');
    });

    it('should show the same values for a specific Standard in "Passing Standards Across Clusters" as displayed in "Controls in Compliance" for that Standard\'s list page', () => {
        cy.get(selectors.widget.passingStandardsAcrossClusters.barLabels)
            .eq(0)
            .invoke('text')
            .then(horizontalBarPassing => {
                cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks)
                    .eq(0)
                    .click();
                cy.get(selectors.widget.controlsInCompliance.centerLabel)
                    .invoke('text')
                    .should('eq', horizontalBarPassing);
            });
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
