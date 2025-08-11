import withAuth from '../../../helpers/basicAuth';

import {
    visitWorkloadCveOverview,
    visitNamespaceView,
    // waitForTableLoadCompleteIndicator,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Namespace View', () => {
    withAuth();

    it('should display the correct search filter chips on the main list page when clicking the deployment link in the table', () => {
        visitWorkloadCveOverview();

        visitNamespaceView();

        // ROX-30492: DOM assertion often, but not always, times out, even with retry on CI
        // for gke but not ocp-4-19 starting about 2025-08-07
        // David observed in video that request finishes ahead of assertion about loading spinner.
        // Instead, wait for request in visit function.
        // waitForTableLoadCompleteIndicator();

        cy.get(selectors.firstTableRow).then(($row) => {
            const namespace = $row.find('td[data-label="Namespace"]').text();
            const cluster = $row.find('td[data-label="Cluster"]').text();

            cy.wrap($row.find('td[data-label="Deployments"] a')).click();

            cy.get(`h1:contains("Platform vulnerabilities")`);

            cy.get(selectors.filterChipGroupItem('Namespace', `^${namespace}$`));
            cy.get(selectors.filterChipGroupItem('Cluster', `^${cluster}$`));
        });
    });
});
