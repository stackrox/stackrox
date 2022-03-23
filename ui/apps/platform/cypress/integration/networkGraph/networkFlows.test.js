import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';
import selectors from '../../selectors';
import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName, selectNamespaceFilters } from '../../helpers/networkGraph';

const tableDataRows = 'table tr[data-testid="data-row"]';
const tableStatusHeaders = 'table tr[data-testid="subhead-row"]';
const sensorTableRow = `${tableDataRows}:contains("sensor")`;
const confirmationButton = 'button:contains("Yes")';
const markAsAnomalousButton = `${sensorTableRow} button:contains("Mark as anomalous")`;
const baselineSettingsTab = `${selectors.tab.tabs}:contains('Baseline Settings')`;

describe('Network Baseline Flows', () => {
    withAuth();
    beforeEach(() => {
        cy.server();

        cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
        cy.route('GET', api.network.networkGraph).as('networkGraph');
        cy.route('GET', api.network.networkBaseline).as('networkBaseline');
        cy.route('POST', api.network.networkBaselineStatus).as('networkBaselineStatus');
        cy.route('PATCH', api.network.networkBaselineLock).as('networkBaselineLock');
        cy.route('PATCH', api.network.networkBaselineUnlock).as('networkBaselineUnlock');

        cy.visit(networkUrl);
        selectNamespaceFilters('stackrox');
        cy.wait('@networkPoliciesGraph');
        cy.wait('@networkGraph');
    });

    describe('Navigating to Deployment', () => {
        it('should navigate to a different deployment when clicking the "Navigate" button', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const tabbedOverlayHeader = '[data-testid="network-entity-tabbed-overlay-header"]';
                const navigateButton = `${sensorTableRow} button:contains("Navigate")`;

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                cy.get(tabbedOverlayHeader).contains('central');

                cy.get(sensorTableRow).trigger('mouseover');
                cy.get(navigateButton).click();

                cy.get(tabbedOverlayHeader).contains('sensor');
            });
        });
    });

    describe('Active Network Flows', () => {
        it('should show anomalous flows section above the baseline flows', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                cy.get(tableStatusHeaders).eq(0).contains('Anomalous Flow');
                cy.get(tableStatusHeaders).eq(1).contains('Baseline Flow');
            });
        });
    });

    describe('Toggling Status of Active Baseline Network Flows', () => {
        it('should be able to toggle status of a single flow', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const addToBaselineButton = `${sensorTableRow} button:contains("Add to baseline")`;

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                // marking a baseline flow as anomalous should show up as anomalous
                cy.get(sensorTableRow).trigger('mouseover');
                cy.get(markAsAnomalousButton).click();
                cy.wait('@networkPoliciesGraph');
                cy.wait('@networkGraph');
                cy.wait('@networkBaselineStatus');
                cy.get(sensorTableRow).trigger('mouseover');
                cy.get(addToBaselineButton);

                // marking an anomalous flow as baseline should show up as baseline
                cy.get(addToBaselineButton).click();
                cy.wait('@networkPoliciesGraph');
                cy.wait('@networkGraph');
                cy.wait('@networkBaselineStatus');
                cy.get(sensorTableRow).trigger('mouseover');
                cy.get(markAsAnomalousButton);
            });
        });

        it('should be able to toggle status of all flows', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const markAllAsAnomalousButton = 'button:contains("Mark all as anomalous")';
                const addAllToBaselineButton = 'button:contains("Add all to baseline")';
                const noAnomalousFlows = 'td:contains("No anomalous flows")';
                const noBaselineFlows = 'td:contains("No baseline flows")';

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                // marking all baseline flows as anomalous should show up as anomalous
                cy.get(markAllAsAnomalousButton).click();
                cy.get(confirmationButton).click();
                cy.wait('@networkPoliciesGraph');
                cy.wait('@networkGraph');
                cy.wait('@networkBaselineStatus');
                cy.get(noBaselineFlows);

                // marking all anomalous flows as baseline should show up as baseline
                cy.get(addAllToBaselineButton).click();
                cy.get(confirmationButton).click();
                cy.wait('@networkPoliciesGraph');
                cy.wait('@networkGraph');
                cy.wait('@networkBaselineStatus');
                cy.get(noAnomalousFlows);
            });
        });

        it('should be able to toggle status of selected flows', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const anomalousStatusHeader = `${tableStatusHeaders}:eq(0)`;
                const sensorTableRowCheckbox = `${sensorTableRow} input[type=checkbox]`;
                const markSelectedAsAnomalousButton = 'button:contains("Mark 1 as anomalous")';
                const addSelectedToBaselineButton = 'button:contains("Add 1 to baseline")';

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                cy.get(anomalousStatusHeader)
                    .invoke('text')
                    .then((anomalousFlowsText) => {
                        // get the number value of anomalous flows
                        const prevNumAnomalousFlows = parseInt(anomalousFlowsText, 10);
                        const prevAnomalousFlowsText = `${tableStatusHeaders}:contains("${anomalousFlowsText}")`;
                        const postAnomalousFlowsText = `${tableStatusHeaders}:contains("${
                            prevNumAnomalousFlows + 1
                        } Anomalous Flow")`;

                        // marking selected baseline flows as anomalous should show up as anomalous
                        cy.get(sensorTableRowCheckbox).check();
                        cy.get(markSelectedAsAnomalousButton).click();
                        cy.get(confirmationButton).click();
                        cy.wait('@networkPoliciesGraph');
                        cy.wait('@networkGraph');
                        cy.wait('@networkBaselineStatus');
                        cy.get(postAnomalousFlowsText);

                        // marking selected anomalous flows as baseline should show up as baseline
                        cy.get(sensorTableRowCheckbox).check();
                        cy.get(addSelectedToBaselineButton).click();
                        cy.get(confirmationButton).click();
                        cy.wait('@networkPoliciesGraph');
                        cy.wait('@networkGraph');
                        cy.wait('@networkBaselineStatus');
                        cy.get(prevAnomalousFlowsText);
                    });
            });
        });
    });

    describe('Baseline Settings', () => {
        it('should not show the anomalous flows section', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });
                cy.get(baselineSettingsTab).click();
                cy.get(tableStatusHeaders).eq(0).contains('Baseline Flow');
            });
        });

        it('should be able to toggle status of a single baseline flow', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const baselineStatusHeader = `${tableStatusHeaders}:eq(0)`;

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });
                cy.get(baselineSettingsTab).click();

                cy.get(baselineStatusHeader)
                    .invoke('text')
                    .then((baselineFlowsText) => {
                        // get the number value of baseline flows
                        const prevNumBaselineFlows = parseInt(baselineFlowsText, 10);
                        const postNumBaselineFlowsText = `${tableStatusHeaders}:contains("${
                            prevNumBaselineFlows - 1
                        } Baseline Flow")`;

                        // marking a baseline flow as anomalous should remove it from the baseline
                        cy.get(sensorTableRow).trigger('mouseover');
                        cy.get(markAsAnomalousButton).click();
                        cy.wait('@networkPoliciesGraph');
                        cy.wait('@networkGraph');
                        cy.wait('@networkBaselineStatus');
                        cy.get(postNumBaselineFlowsText);
                    });
            });
        });

        describe('Cluster with Helm management', () => {
            it('should toggle the alert on baseline violations toggle', () => {
                cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                    const baselineViolationsToggle = '[data-testid="toggle-switch-checkbox"]';
                    clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });
                    cy.get(baselineSettingsTab).click();
                    cy.get(baselineViolationsToggle).should('not.be.checked');
                    cy.get(baselineViolationsToggle).check({ force: true });
                    cy.wait('@networkBaselineLock');
                    cy.wait('@networkBaseline');
                    cy.get(baselineViolationsToggle).should('be.checked');
                    cy.get(baselineViolationsToggle).uncheck({ force: true });
                    cy.wait('@networkBaselineUnlock');
                    cy.wait('@networkBaseline');
                    cy.get(baselineViolationsToggle).should('not.be.checked');
                });
            });
        });
    });
});
