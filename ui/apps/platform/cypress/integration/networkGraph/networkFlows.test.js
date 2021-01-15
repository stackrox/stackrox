import { url as networkUrl, selectors as networkPageSelectors } from '../../constants/NetworkPage';

import * as api from '../../constants/apiEndpoints';
import withAuth from '../../helpers/basicAuth';
import { clickOnNodeByName } from '../../helpers/networkGraph';

describe('Network Baseline Flows', () => {
    withAuth();
    beforeEach(() => {
        cy.server();

        cy.route('GET', api.network.networkPoliciesGraph).as('networkPoliciesGraph');
        cy.route('GET', api.network.networkGraph).as('networkGraph');
        cy.route('POST', api.network.networkBaselineStatus).as('networkBaselineStatus');

        cy.visit(networkUrl);
        cy.wait('@networkPoliciesGraph');
        cy.wait('@networkGraph');
    });

    describe('Active Network Flows', () => {
        it('should show anomalous flows section above the baseline flows', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const statusHeaders = '[data-testid="subhead-row"]';

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                cy.get(statusHeaders).eq(0).contains('Anomalous Flow');
                cy.get(statusHeaders).eq(1).contains('Baseline Flow');
            });
        });
    });

    describe('Toggling Status of Active Baseline Network Flows', () => {
        it('should be able to toggle status of a single flow', () => {
            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                const sensorTableRow = 'table [data-testid="data-row"]:contains("sensor")';
                const markAsAnomalousButton = `${sensorTableRow} button:contains("Mark as anomalous")`;
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
                const confirmationButton = 'button:contains("Yes")';
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
                const anomalousStatusHeader = 'table tr[data-testid="subhead-row"]:eq(0)';
                const confirmationButton = 'button:contains("Yes")';
                const sensorTableRowCheckbox = 'table tr:contains("sensor") input[type=checkbox]';
                const markSelectedAsAnomalousButton = 'button:contains("Mark 1 as anomalous")';
                const addSelectedToBaselineButton = 'button:contains("Add 1 to baseline")';

                clickOnNodeByName(cytoscape, { type: 'DEPLOYMENT', name: 'central' });

                cy.get(anomalousStatusHeader)
                    .invoke('text')
                    .then((anomalousFlowsText) => {
                        // get the number value of anomalous flows
                        const prevNumAnomalousFlows = parseInt(anomalousFlowsText, 10);
                        const prevAnomalousFlowsText = `table tr[data-testid="subhead-row"]:contains("${anomalousFlowsText}")`;
                        const postAnomalousFlowsText = `table tr[data-testid="subhead-row"]:contains("${
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
});
