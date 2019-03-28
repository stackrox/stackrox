import { url, selectors } from './constants/CompliancePage';
import withAuth from './helpers/basicAuth';

describe('Compliance dashboard page', () => {
    withAuth();

    beforeEach(() => {
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
                cy.get(selectors.list.table.rows)
                    .its('length')
                    .should('eq', numClusters);
            });
    });

    // TODO(ROX-1774): Fix and re-enable
    xit('should show the same amount of namespaces between the Dashboard and List Page', () => {
        cy.get(selectors.dashboard.tileLinks.namespace.value)
            .invoke('text')
            .then(text => {
                const numNamespaces = Number(text);
                cy.visit(url.list.namespaces);
                cy.get(selectors.list.table.rows)
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
                cy.get(selectors.list.table.rows)
                    .its('length')
                    .should('eq', numNodes);
            });
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

    it('should link from Passing Standards Across Clusters widget to standards grouped by clusters list', () => {
        cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks)
            .first()
            .click();
        cy.url().should('contain', '?groupBy=CLUSTER');
        cy.get(selectors.list.table.header).contains('CLUSTER');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Namespaces widget to standards grouped by namespaces list', () => {
        cy.get(selectors.widget.passingStandardsAcrossNamespaces.axisLinks)
            .first()
            .click();
        cy.url().should('contain', '?groupBy=NAMESPACE');
        cy.get(selectors.list.table.header).contains('NAMESPACE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Nodes widget to standards grouped by nodes list', () => {
        cy.get(selectors.widget.passingStandardsAcrossNodes.axisLinks)
            .first()
            .click();
        cy.url().should('contain', '?groupBy=NODE');
        cy.get(selectors.list.table.header).contains('NODE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link to controls list when clicking on "# controls" in sunburst', () => {
        cy.get(selectors.widget.PCICompliance.controls)
            .first()
            .click();
        cy.location('pathname').should('eq', url.list.standards.PCI_DSS_3_2);
    });
});
