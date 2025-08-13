import withAuth from '../../../helpers/basicAuth';

import {
    interactAndWaitForDeploymentList,
    visitWorkloadCveOverview,
    visitNamespaceView,
} from './WorkloadCves.helpers';
import { selectors } from './WorkloadCves.selectors';

describe('Workload CVE Namespace View', () => {
    withAuth();

    it('should display the correct search filter chips on the main list page when clicking the deployment link in the table', () => {
        visitWorkloadCveOverview();

        visitNamespaceView();

        // ROX-30503: need to wait for each table cell to exist in the DOM.
        const namespaceSelector = `${selectors.firstTableRow} td[data-label="Namespace"]`;
        const clusterSelector = `${selectors.firstTableRow} td[data-label="Cluster"]`;
        const deploymentsLinkSelector = `${selectors.firstTableRow} td[data-label="Deployments"] a`;

        cy.get(namespaceSelector)
            .invoke('text')
            .then((namespace) => {
                cy.get(clusterSelector)
                    .invoke('text')
                    .then((cluster) => {
                        interactAndWaitForDeploymentList(() => {
                            cy.get(deploymentsLinkSelector).click();
                        });

                        cy.get(`h1:contains("Platform vulnerabilities")`);

                        cy.get(selectors.filterChipGroupItem('Namespace', `^${namespace}$`));
                        cy.get(selectors.filterChipGroupItem('Cluster', `^${cluster}$`));
                    });
            });
    });
});
