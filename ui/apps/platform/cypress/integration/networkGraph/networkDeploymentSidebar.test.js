import withAuth from '../../helpers/basicAuth';
import { hasOrchestratorFlavor } from '../../helpers/features';

import {
    visitNetworkGraph,
    checkNetworkGraphEmptyState,
    selectCluster,
    selectNamespace,
    selectDeployment,
} from './networkGraph.helpers';
import { networkGraphSelectors } from './networkGraph.selectors';

describe('Network Graph deployment sidebar', () => {
    withAuth();

    it('should render a graph when cluster and namespace are selected', function () {
        // Skipping this test due to an issue in GKE-latest that is not easily fixable, and will
        // cause failures 100% of the time. See ROX-23907, ROX-22431.
        // TODO - remove this skip https://issues.redhat.com/browse/ROX-25038
        if (!hasOrchestratorFlavor('openshift')) {
            this.skip();
        }

        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');
        selectDeployment('collector');

        // confirm that the graph only contains collector and other StackRox deployments it communiticates with
        cy.get(
            `${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label:contains("sensor")`
        );
        cy.get(
            `${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label:contains("central")`
        ).should('not.exist');
        cy.get(
            `${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label:contains("scanner")`
        ).should('not.exist');
        cy.get(
            `${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label:contains("admission-controller")`
        ).should('not.exist');

        // click on Collector node, too
        cy.get(`${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label`)
            .contains('collector')
            .parent()
            .click();

        cy.get(networkGraphSelectors.drawerTitle).contains('collector');
        cy.get(`${networkGraphSelectors.drawerSubtitle}:contains("stackrox")`); // cluster name flexible for any test environment

        // check Details tab
        cy.get(`${networkGraphSelectors.drawerTabs} .pf-m-current`).contains('Details');

        cy.get(
            '.pf-v5-c-expandable-section .pf-v5-c-expandable-section__toggle-text:contains("Network security")'
        );
        cy.get(
            '.pf-v5-c-expandable-section .pf-v5-c-expandable-section__toggle-text:contains("Deployment overview")'
        );
        cy.get(
            '.pf-v5-c-expandable-section .pf-v5-c-expandable-section__toggle-text:contains("Port configurations")'
        );
        cy.get(
            '.pf-v5-c-expandable-section .pf-v5-c-expandable-section__toggle-text:contains("Container configurations")'
        );

        // check list of containers in Container Config section
        cy.get('.pf-v5-c-expandable-section:contains("Container configurations")').find(
            '[data-testid="deployment-container-config"] .pf-v5-c-expandable-section__toggle-text:contains("collector")'
        );
        cy.get('.pf-v5-c-expandable-section:contains("Container configurations")').find(
            '[data-testid="deployment-container-config"] .pf-v5-c-expandable-section__toggle-text:contains("compliance")'
        );
    });
});
