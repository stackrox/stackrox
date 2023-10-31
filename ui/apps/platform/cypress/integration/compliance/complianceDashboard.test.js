import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    interactAndWaitForComplianceStandard,
    triggerScan,
    verifyDashboardEntityLink,
    visitComplianceDashboard,
} from './Compliance.helpers';
import { selectors } from './Compliance.selectors';

function getWidgetSelector(headerText) {
    return `.widget:has([data-testid="widget-header"]:contains("${headerText}"))`;
}

function getStandardAcrossEntitiesLink(entityNameOrdinaryCasePlural) {
    return `${getWidgetSelector(`Passing standards across ${entityNameOrdinaryCasePlural}`)} a`;
}

describe('Compliance Dashboard', () => {
    withAuth();

    it('should scan for compliance data', () => {
        triggerScan(); // prerequisite for the following tests
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
            cy.get(getStandardAcrossEntitiesLink('clusters')).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=CLUSTER'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('cluster');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Namespaces widget to standards grouped by namespaces list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(getStandardAcrossEntitiesLink('namespaces')).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NAMESPACE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('namespace');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });

    it('should link from Passing Standards Across Nodes widget to standards grouped by nodes list', () => {
        visitComplianceDashboard();

        interactAndWaitForComplianceStandard(() => {
            cy.get(getStandardAcrossEntitiesLink('nodes')).first().click();
        });
        cy.location('search').should('contain', '?s[groupBy]=NODE'); // followed by a standard
        cy.get('[data-testid="panel-header"]').contains('node');
        cy.get(selectors.list.table.firstGroup).should('be.visible');
    });
});
