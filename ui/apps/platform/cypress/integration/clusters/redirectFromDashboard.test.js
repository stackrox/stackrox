import withAuth from '../../helpers/basicAuth';
import { summaryCountsOpname, visitMainDashboard } from '../../helpers/main';

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
            const staticResponseMapForDashboard = {
                [summaryCountsOpname]: {
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
                },
            };
            visitMainDashboard(staticResponseMapForDashboard);
        }, staticResponseMapForClusters);
    });
});
