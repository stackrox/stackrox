import { hasFeatureFlag } from '../../../helpers/features';
import withAuth from '../../../helpers/basicAuth';

import {
    visitWorkloadCveOverview,
    visitNamespaceView,
    waitForTableLoadCompleteIndicator,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Namespace View', () => {
    withAuth();

    it('should display the correct search filter chips on the main list page when clicking the deployment link in the table', () => {
        visitWorkloadCveOverview();

        visitNamespaceView();

        waitForTableLoadCompleteIndicator();

        cy.get(selectors.firstTableRow).then(($row) => {
            const namespace = $row.find('td[data-label="Namespace"]').text();
            const cluster = $row.find('td[data-label="Cluster"]').text();

            cy.wrap($row.find('td[data-label="Deployments"] a')).click();

            const pageTitle = hasFeatureFlag('ROX_PLATFORM_CVE_SPLIT')
                ? 'Platform vulnerabilities'
                : 'Workload CVEs';
            cy.get(`h1:contains("${pageTitle}")`);

            cy.get(selectors.filterChipGroupItem('Namespace', `^${namespace}$`));
            cy.get(selectors.filterChipGroupItem('Cluster', `^${cluster}$`));
        });
    });
});
