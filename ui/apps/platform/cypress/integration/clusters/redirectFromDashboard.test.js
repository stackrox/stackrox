import withAuth from '../../helpers/basicAuth';
import { visitMainDashboardWithStaticResponseForSummaryCounts } from '../../helpers/main';

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
            const staticResponseForSummaryCounts = {
                body: {
                    data: {
                        clusterCount: 0, // no secured clusters
                        nodeCount: 3,
                        violationCount: 20,
                        deploymentCount: 35,
                        imageCount: 31,
                        secretCount: 15,
                    },
                },
            };
            visitMainDashboardWithStaticResponseForSummaryCounts(staticResponseForSummaryCounts);
        }, staticResponseMapForClusters);

        cy.get(
            'p:contains("You have successfully deployed a Red Hat Advanced Cluster Security platform.")'
        );

        cy.get('h2:contains("Configure the clusters you want to secure.")');

        cy.get('a:contains("View instructions")');
    });
});
