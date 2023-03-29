import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import {
    clickBaselineSettingsTab,
    clickOnDeploymentNodeByName,
    interactAndWaitForChangeToNetworkFlows,
    visitOldNetworkGraphWithNamespaceFilter,
} from '../../helpers/networkGraph';

const tableStatusHeaders = 'table tr[data-testid="subhead-row"]';

const tableDataRows = 'table tr[data-testid="data-row"]';
const sensorTableRow = `${tableDataRows}:contains("sensor")`;

// Too bad, so sad: Anomalous Flows and Baseline Flows are not in separate tbody elements.

const sensorTableRowUnderAnomalousFlows = `${tableDataRows}.bg-alert-200:contains("sensor")`;
const addToBaselineButton = `${sensorTableRowUnderAnomalousFlows} button:contains("Add to baseline")`;

const sensorTableRowUnderBaselineFlows = `${tableDataRows}.bg-base-100:contains("sensor")`;
const markAsAnomalousButton = `${sensorTableRowUnderBaselineFlows} button:contains("Mark as anomalous")`;

function clickFlowsConfirmationButton() {
    interactAndWaitForChangeToNetworkFlows(() => {
        cy.get(networkPageSelectors.buttons.confirmationButton).click();
    });
}

describe('Network Baseline Flows', () => {
    withAuth();

    describe('Navigating to Deployment', () => {
        it('should navigate to a different deployment when clicking the "Navigate" button', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const tabbedOverlayHeader = '[data-testid="network-entity-tabbed-overlay-header"]';
                const navigateButton = `${sensorTableRow} button:contains("Navigate")`;

                clickOnDeploymentNodeByName(cytoscape, 'central');

                cy.get(tabbedOverlayHeader).contains('central');

                cy.get(sensorTableRow).trigger('mouseover');
                cy.get(navigateButton).click({ force: true }); // because network-zoom-buttons can cover it

                cy.get(tabbedOverlayHeader).contains('sensor');
            });
        });
    });

    describe('Active Network Flows', () => {
        it('should show table column headings', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnDeploymentNodeByName(cytoscape, 'central');

                cy.get('th:contains("Entity")');
                cy.get('th:contains("Traffic")');
                cy.get('th:contains("Type")');
                cy.get('th:contains("Namespace")');
                cy.get('th:contains("State")');
                cy.get('th:contains("Protocol")');
                cy.get('th:contains("Port")');
            });
        });

        it('should show anomalous flows section above the baseline flows', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnDeploymentNodeByName(cytoscape, 'central');

                cy.get(tableStatusHeaders).eq(0).contains('Anomalous Flow');
                cy.get(tableStatusHeaders).eq(1).contains('Baseline Flow');
            });
        });
    });

    describe('Toggling Status of Active Baseline Network Flows', () => {
        it('should be able to toggle status of a single flow', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnDeploymentNodeByName(cytoscape, 'central');

                // marking a baseline flow as anomalous should show up as anomalous
                cy.get(sensorTableRowUnderBaselineFlows).trigger('mouseover');
                interactAndWaitForChangeToNetworkFlows(() => {
                    cy.get(markAsAnomalousButton).click({ force: true }); // because network-zoom-buttons can cover it
                });

                // marking an anomalous flow as baseline should show up as baseline
                cy.get(sensorTableRowUnderAnomalousFlows).trigger('mouseover');
                interactAndWaitForChangeToNetworkFlows(() => {
                    cy.get(addToBaselineButton).click({ force: true }); // because network-zoom-buttons can cover it
                });

                // Assert that test leaves table row in its initial state.
                cy.get(sensorTableRowUnderBaselineFlows).trigger('mouseover');
                cy.get(markAsAnomalousButton);
            });
        });

        it('should be able to toggle status of all flows', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const markAllAsAnomalousButton = 'button:contains("Mark all as anomalous")';
                const addAllToBaselineButton = 'button:contains("Add all to baseline")';
                const noAnomalousFlows = 'td:contains("No anomalous flows")';
                const noBaselineFlows = 'td:contains("No baseline flows")';

                clickOnDeploymentNodeByName(cytoscape, 'central');

                // marking all baseline flows as anomalous should show up as anomalous
                cy.get(markAllAsAnomalousButton).click();
                clickFlowsConfirmationButton();
                cy.get(noBaselineFlows);

                // marking all anomalous flows as baseline should show up as baseline
                cy.get(addAllToBaselineButton).click();
                clickFlowsConfirmationButton();
                cy.get(noAnomalousFlows);

                // The following test seems to depend the result of this test.
            });
        });

        it('should be able to toggle status of selected flows', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const anomalousStatusHeader = `${tableStatusHeaders}:eq(0)`;
                const sensorTableRowCheckbox = `${sensorTableRow} input[type=checkbox]`;
                const markSelectedAsAnomalousButton = 'button:contains("Mark 1 as anomalous")';
                const addSelectedToBaselineButton = 'button:contains("Add 1 to baseline")';

                clickOnDeploymentNodeByName(cytoscape, 'central');

                cy.get(anomalousStatusHeader)
                    .invoke('text')
                    .then((anomalousFlowsText) => {
                        // get the number value of anomalous flows
                        const prevNumAnomalousFlows = parseInt(anomalousFlowsText, 10);
                        const prevAnomalousFlowsText = `${tableStatusHeaders}:contains("${anomalousFlowsText}")`;
                        const postAnomalousFlowsText = `${tableStatusHeaders}:contains("${
                            prevNumAnomalousFlows + 1
                        } Anomalous Flow")`;

                        // This test seems to depend the result of the preceding test.

                        // marking selected baseline flows as anomalous should show up as anomalous
                        cy.get(sensorTableRowCheckbox).check();
                        cy.get(markSelectedAsAnomalousButton).click();
                        clickFlowsConfirmationButton();
                        cy.get(postAnomalousFlowsText);

                        // marking selected anomalous flows as baseline should show up as baseline
                        cy.get(sensorTableRowCheckbox).check();
                        cy.get(addSelectedToBaselineButton).click();
                        clickFlowsConfirmationButton();
                        cy.get(prevAnomalousFlowsText);
                    });
            });
        });
    });

    describe('Baseline Settings', () => {
        it('should not show the anomalous flows section', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnDeploymentNodeByName(cytoscape, 'central');

                clickBaselineSettingsTab();

                cy.get(tableStatusHeaders).eq(0).contains('Baseline Flow');
            });
        });

        it('should be able to toggle status of a single baseline flow', () => {
            visitOldNetworkGraphWithNamespaceFilter('stackrox');

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const baselineStatusHeader = `${tableStatusHeaders}:eq(0)`;

                clickOnDeploymentNodeByName(cytoscape, 'central');

                clickBaselineSettingsTab();

                cy.get(baselineStatusHeader)
                    .invoke('text')
                    .then((baselineFlowsText) => {
                        // get the number value of baseline flows
                        const prevNumBaselineFlows = parseInt(baselineFlowsText, 10);
                        const postNumBaselineFlowsText = `${tableStatusHeaders}:contains("${
                            prevNumBaselineFlows - 1
                        } Baseline Flow")`;

                        // marking a baseline flow as anomalous should remove it from the baseline
                        cy.intercept('PATCH', api.network.networkBaselinePeers).as(
                            'networkBaselinePeers'
                        );
                        cy.intercept('GET', api.network.networkGraph).as('networkGraph');
                        cy.intercept('GET', api.network.networkPoliciesGraph).as(
                            'networkPoliciesGraph'
                        );
                        cy.get(sensorTableRow).trigger('mouseover');
                        cy.get(markAsAnomalousButton).click({ force: true }); // because network-zoom-buttons can cover it
                        cy.wait([
                            '@networkBaselinePeers',
                            '@networkGraph',
                            '@networkPoliciesGraph',
                        ]);

                        cy.get(postNumBaselineFlowsText);
                    });
            });
        });

        describe('Cluster with Helm management', () => {
            it('should toggle the alert on baseline violations toggle', () => {
                visitOldNetworkGraphWithNamespaceFilter('stackrox');

                cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                    clickOnDeploymentNodeByName(cytoscape, 'central');

                    clickBaselineSettingsTab();

                    const baselineViolationsToggle = '[data-testid="toggle-switch-checkbox"]';

                    cy.get(baselineViolationsToggle).should('not.be.checked');

                    cy.intercept('PATCH', api.network.networkBaselineLock).as(
                        'networkBaselineLock'
                    );
                    cy.get(baselineViolationsToggle).check({ force: true });
                    cy.wait('@networkBaselineLock');

                    cy.get(baselineViolationsToggle).should('be.checked');
                    cy.intercept('PATCH', api.network.networkBaselineUnlock).as(
                        'networkBaselineUnlock'
                    );
                    cy.get(baselineViolationsToggle).uncheck({ force: true });
                    cy.wait('@networkBaselineUnlock');

                    cy.get(baselineViolationsToggle).should('not.be.checked');
                });
            });
        });
    });
});
