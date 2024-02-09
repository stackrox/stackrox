import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
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

        if (hasFeatureFlag('ROX_MOVE_INIT_BUNDLES_UI')) {
            // Replace h1 with h2 if we factor out a minimal Clusters page.
            cy.get('h1:contains("Secure clusters with a reusable init bundle")');
            // Button text depends whether or not init bundles exist.
            cy.get('button.pf-m-primary.pf-m-display-lg');
        } else {
            cy.get('h2:contains("Configure the clusters you want to secure.")');
            cy.get('a:contains("View instructions")');
        }
    });
});
