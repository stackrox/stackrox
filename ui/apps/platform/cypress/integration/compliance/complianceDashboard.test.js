import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    interactAndWaitForComplianceEntities,
    interactAndWaitForComplianceStandard,
    scanCompliance,
    visitComplianceDashboard,
} from './Compliance.helpers';
import { selectors } from './Compliance.selectors';

describe('Compliance Dashboard', () => {
    withAuth();

    it('should scan for compliance data', () => {
        visitComplianceDashboard();
        scanCompliance(); // prerequisite for the following tests
    });

    it('should have title', () => {
        visitComplianceDashboard();

        cy.title().should('match', getRegExpForTitleWithBranding('Compliance'));
    });

    it('should show the same amount of clusters as list', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.cluster.value)
            .invoke('text')
            .then((text) => {
                const count = parseInt(text, 10); // for example, 1 cluster
                interactAndWaitForComplianceEntities(() => {
                    cy.get(selectors.dashboard.tileLinks.cluster.tile).click();
                }, 'clusters');
                cy.get(`[data-testid="panel-header"]:contains("${count}")`);
            });
    });

    it('should show the same amount of namespaces as list', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.namespace.value)
            .invoke('text')
            .then((text) => {
                const count = parseInt(text, 10); // for example, 2 namespaces
                interactAndWaitForComplianceEntities(() => {
                    cy.get(selectors.dashboard.tileLinks.namespace.tile).click();
                }, 'namespaces');
                cy.get(`[data-testid="panel-header"]:contains("${count}")`);
            });
    });

    it('should show the same amount of nodes as list', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.node.value)
            .invoke('text')
            .then((text) => {
                const count = parseInt(text, 10); // for example, 2 nodes
                interactAndWaitForComplianceEntities(() => {
                    cy.get(selectors.dashboard.tileLinks.node.tile).click();
                }, 'nodes');
                cy.get(`[data-testid="panel-header"]:contains("${count}")`);
            });
    });

    it('should show the same amount of deployments as list', () => {
        visitComplianceDashboard();

        cy.get(selectors.dashboard.tileLinks.deployment.value)
            .invoke('text')
            .then((text) => {
                const count = parseInt(text, 10); // for example, 2 deployments
                interactAndWaitForComplianceEntities(() => {
                    cy.get(selectors.dashboard.tileLinks.deployment.tile).click();
                }, 'deployments');
                cy.get(`[data-testid="panel-header"]:contains("${count}")`);
            });
    });

    it('should link from Passing Standards Across Clusters widget to standards grouped by clusters list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=CLUSTER'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('CLUSTER');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Namespaces widget to standards grouped by namespaces list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossNamespaces.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NAMESPACE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('NAMESPACE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Nodes widget to standards grouped by nodes list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossNodes.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NODE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('NODE');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link to controls list when clicking on "# controls" in sunburst', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.PCICompliance.controls).first().click();
        });
        cy.location('search').should('eq', '?s[standard]=PCI DSS 3.2.1'.replace(/ /g, '%20'));
    });
});
