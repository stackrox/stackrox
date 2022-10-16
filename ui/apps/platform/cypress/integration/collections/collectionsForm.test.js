import { selectors } from '../../constants/CompliancePage';
import withAuth from '../../helpers/basicAuth';
import { visitComplianceDashboard, visitComplianceEntities } from '../../helpers/compliance';

describe('Compliance dashboard page', () => {
    withAuth();

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
});
