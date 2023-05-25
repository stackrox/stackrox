import withAuth from '../../helpers/basicAuth';
import { hasOrchestratorFlavor } from '../../helpers/features';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    interactAndWaitForComplianceStandard,
    scanCompliance,
    verifyDashboardEntityLink,
    visitComplianceDashboard,
} from './Compliance.helpers';
import { selectors } from './Compliance.selectors';

describe('Compliance Dashboard', () => {
    withAuth();

    before(function beforeHook() {
        if (hasOrchestratorFlavor('openshift')) {
            this.skip();
        }
    });

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

        verifyDashboardEntityLink('clusters', /^\d+ clusters?/); // include ^ but omit $
    });

    it('should show the same amount of namespaces as list', () => {
        visitComplianceDashboard();

        verifyDashboardEntityLink('namespaces', /^\d+ namespaces?/); // include ^ but omit $
    });

    it('should show the same amount of nodes as list', () => {
        visitComplianceDashboard();

        verifyDashboardEntityLink('nodes', /^\d+ nodes?/); // include ^ but omit $
    });

    it('should show the same amount of deployments as list', () => {
        visitComplianceDashboard();

        verifyDashboardEntityLink('deployments', /^\d+ deployments?/); // include ^ but omit $
    });

    it('should link from Passing Standards Across Clusters widget to standards grouped by clusters list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossClusters.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=CLUSTER'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('cluster');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Namespaces widget to standards grouped by namespaces list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossNamespaces.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NAMESPACE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('namespace');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Nodes widget to standards grouped by nodes list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(selectors.widget.passingStandardsAcrossNodes.axisLinks).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NODE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('node');
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
