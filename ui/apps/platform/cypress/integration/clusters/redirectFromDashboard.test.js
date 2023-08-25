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

        cy.get(
            'p:contains("You have successfully deployed a Red Hat Advanced Cluster Security platform.")'
        );

        cy.get('h2:contains("Configure the clusters you want to secure.")');

        cy.get('a:contains("View instructions")');
    });
});
