import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import { visitClusters, visitClustersFromLeftNav } from './Clusters.helpers';

describe('Clusters', () => {
    withAuth();

    it('should visit via link in left nav', () => {
        visitClustersFromLeftNav();
    });

    it('should display title and columns expected in clusters list page', () => {
        visitClusters();

        cy.title().should('match', getRegExpForTitleWithBranding('Clusters'));

        cy.get('th:contains("Cluster")');
        cy.get('th:contains("Provider (Region)")');
        cy.get('th:contains("Cluster status")');
        cy.get('th:contains("Sensor compatibility status")');
        cy.get('th:contains("Credential expiration")');
        // Cluster deletion: see clusterDeletion.test.js
    });
});
