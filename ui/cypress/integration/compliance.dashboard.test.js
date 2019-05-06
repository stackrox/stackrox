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
});
