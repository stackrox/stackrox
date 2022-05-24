import { selectors } from '../../constants/CompliancePage';
import withAuth from '../../helpers/basicAuth';
import {
    scanCompliance,
    visitComplianceDashboard,
    visitComplianceEntities,
} from '../../helpers/compliance';

describe('Compliance dashboard page', () => {
    withAuth();

    it('should scan for compliance data from the Dashboard page', () => {
        visitComplianceDashboard();
        scanCompliance(); // prerequisite for the following tests
    });

    it('should show the same amount of clusters between the Dashboard and List Page', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.cluster.value)
            .invoke('text')
            .then((text) => {
                const numClusters = parseInt(text, 10); // for example, 1 cluster
                visitComplianceEntities('clusters');
                cy.get(selectors.list.table.rows).its('length').should('eq', numClusters);
            });
    });

    it('should show the same amount of namespaces between the Dashboard and List Page', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.namespace.value)
            .invoke('text')
            .then((text) => {
                const numNamespaces = parseInt(text, 10); // for example, 2 namespaces
                visitComplianceEntities('namespaces');
                cy.get(selectors.list.table.rows).its('length').should('eq', numNamespaces);
            });
    });

    it('should show the same amount of nodes between the Dashboard and List Page', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.node.value)
            .invoke('text')
            .then((text) => {
                const numNodes = parseInt(text, 10); // for example, 2 nodes
                visitComplianceEntities('nodes');
                cy.get(selectors.list.table.rows).its('length').should('eq', numNodes);
            });
    });

    it('should link from Passing Standards Across Clusters widget to standards grouped by clusters list', () => {
        visitComplianceDashboard();

        cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks).first().click();
        cy.location('search').should('contain', '?s[groupBy]=CLUSTER'); // followed by a standard
        cy.get(selectors.list.table.header).contains('CLUSTER');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Namespaces widget to standards grouped by namespaces list', () => {
        visitComplianceDashboard();

        cy.get(selectors.widget.passingStandardsAcrossNamespaces.axisLinks).first().click();
        cy.location('search').should('contain', '?s[groupBy]=NAMESPACE'); // followed by a standard
        cy.get(selectors.list.table.header).contains('NAMESPACE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Nodes widget to standards grouped by nodes list', () => {
        visitComplianceDashboard();

        cy.get(selectors.widget.passingStandardsAcrossNodes.axisLinks).first().click();
        cy.location('search').should('contain', '?s[groupBy]=NODE'); // followed by a standard
        cy.get(selectors.list.table.header).contains('NODE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link to controls list when clicking on "# controls" in sunburst', () => {
        visitComplianceDashboard();

        cy.get(selectors.widget.PCICompliance.controls).first().click();
        cy.location('search').should('eq', '?s[standard]=PCI DSS 3.2.1'.replace(/ /g, '%20'));
    });
});
