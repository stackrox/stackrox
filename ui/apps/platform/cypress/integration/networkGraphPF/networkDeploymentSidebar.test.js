import { networkGraphSelectors } from './networkGraph.selectors';

import withAuth from '../../helpers/basicAuth';
import {
    visitNetworkGraph,
    checkNetworkGraphEmptyState,
    selectCluster,
    selectNamespace,
    selectDeployment,
} from '../../helpers/networkGraphPF';
import { hasFeatureFlag } from '../../helpers/features';

describe('Network Graph deployment sidebar', () => {
    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_NETWORK_GRAPH_PATTERNFLY')) {
            this.skip();
        }
    });

    withAuth();

    it('should show deployment details in the sidebar', () => {
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
            '.pf-c-expandable-section .pf-c-expandable-section__toggle-text:contains("Network security")'
        );
        cy.get(
            '.pf-c-expandable-section .pf-c-expandable-section__toggle-text:contains("Deployment overview")'
        );
        cy.get(
            '.pf-c-expandable-section .pf-c-expandable-section__toggle-text:contains("Port configurations")'
        );
        cy.get(
            '.pf-c-expandable-section .pf-c-expandable-section__toggle-text:contains("Container configurations")'
        );

        // check list of containers in Container Config section
        cy.get('.pf-c-expandable-section:contains("Container configurations")').find(
            '[data-testid="deployment-container-config"] .pf-c-expandable-section__toggle-text:contains("collector")'
        );
        cy.get('.pf-c-expandable-section:contains("Container configurations")').find(
            '[data-testid="deployment-container-config"] .pf-c-expandable-section__toggle-text:contains("compliance")'
        );
    });

    it('should show anomalous and baseline flows in the sidebar', () => {
        cy.intercept('POST', '/v1/networkbaseline/*/status').as('networkBaselines');

        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');
        selectDeployment('sensor');

        // click on Collector node, too
        cy.get(`${networkGraphSelectors.nodes} > [data-type="node"] .pf-topology__node__label`)
            .contains('sensor')
            .parent()
            .click();

        // check Flows tab
        cy.get(`${networkGraphSelectors.drawerTabs}`).contains('Flows').click();

        cy.wait('@networkBaselines');

        cy.get(`${networkGraphSelectors.drawerTabs} .pf-m-current:contains("Flows")`); // now that it is clicked, make sure it is selected

        // check breakdown of flows
        // TODO: clean this callback waterfall up
        cy.get('[data-testid="flows-table-header"]')
            .invoke('text')
            .then((allFlowsText) => {
                // get the number value of all flows
                const allFlowsCount = parseInt(allFlowsText, 10);
                cy.get('[data-testid="anomalous-flows-header"]')
                    .invoke('text')
                    .then((anomalousFlowsText) => {
                        // get the number value of anomalous flows
                        const anomalousFlowsCount = parseInt(anomalousFlowsText, 10);
                        cy.get('[data-testid="baseline-flows-header"]')
                            .invoke('text')
                            .then((baselineFlowsText) => {
                                // get the number value of baseline flows
                                const baselineFlowsCount = parseInt(baselineFlowsText, 10);

                                expect(allFlowsCount).to.equal(
                                    anomalousFlowsCount + baselineFlowsCount
                                );
                            });
                    });
            });
    });
});
