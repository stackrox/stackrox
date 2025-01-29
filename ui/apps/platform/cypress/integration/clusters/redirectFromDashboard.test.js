import withAuth from '../../helpers/basicAuth';
import { visitMainDashboardWithStaticResponseForClustersForPermission } from '../../helpers/main';

import { clustersAlias, interactAndVisitClusters } from './Clusters.helpers';

describe('Clusters', () => {
    withAuth();

    it('should redirect from Dashboard when no secured clusters have been added (only applies to Cloud Service)', () => {
        const staticResponseMapForClusters = {
            [clustersAlias]: {
                body: {
                    clusters: [], // no secured clusters
                },
            },
        };

        interactAndVisitClusters(() => {
            const staticResponseForClustersForPermissions = {
                body: {
                    clusters: [], // no secured clusters
                },
            };
            visitMainDashboardWithStaticResponseForClustersForPermission(
                staticResponseForClustersForPermissions
            );
        }, staticResponseMapForClusters);

        // Replace with h2 if refactoring restores h1 element with Clusters
        cy.get('h1:contains("Secure clusters with a reusable init bundle")');
    });
});
