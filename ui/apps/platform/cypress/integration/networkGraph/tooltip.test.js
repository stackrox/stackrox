import * as api from '../../constants/apiEndpoints';
import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import withAuth from '../../helpers/basicAuth';
import {
    clickOnNodeByName,
    mouseOverNodeByName,
    visitNetworkGraphWithMockedData,
} from '../../helpers/networkGraph';
import selectors from '../../selectors/index';

const { cytoscapeContainer } = networkPageSelectors;

// TODO: update mock data to reflect the new Anomalous/Baseline Flows of Network Detection
describe.skip('Network Graph tooltip', () => {
    withAuth();

    describe('deployment node', () => {
        const {
            table: { cells: cellsSelector },
        } = selectors;
        const ingressSelector = `${cellsSelector}:contains("ingress")`;
        const egressSelector = `${cellsSelector}:contains("egress")`;
        const bidirectionalSelector = `${cellsSelector}:contains("bidirectional")`;

        const getIngressFlowsText = (count) => `${count} ingress flows`;
        const getEgressFlowsText = (count) => `${count} egress flows`;

        // TODO: re-enable when https://stack-rox.atlassian.net/browse/ROX-5904 is fixed
        it.skip('has no bidirectional', () => {
            cy.intercept('GET', api.network.deployment, {
                fixture: 'network/sensorDeployment.json',
            }).as('sensorDeployment');
            visitNetworkGraphWithMockedData();

            const name = 'sensor';

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, {
                    type: 'DEPLOYMENT',
                    name,
                });
                cy.wait('@sensorDeployment');

                cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);
                cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                    const nIngressOnly = $trs.has(ingressSelector).length;
                    const nEgressOnly = $trs.has(egressSelector).length;
                    const nBidirectional = $trs.has(bidirectionalSelector).length;
                    expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                    expect(nBidirectional).to.equal(2);

                    cy.get('#panel-close-button').click();

                    cy.getCytoscape(cytoscapeContainer).then((cytoscape2) => {
                        mouseOverNodeByName(cytoscape2, { type: 'DEPLOYMENT', name });

                        cy.get(selectors.tooltip.title).should('contain', name);
                        cy.get(selectors.tooltip.body)
                            .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                            .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                    });
                });
            });
        });

        it('has bidirectional', () => {
            cy.intercept('GET', api.network.deployment, {
                fixture: 'network/centralDeployment.json',
            }).as('centralDeployment');
            visitNetworkGraphWithMockedData();

            const name = 'central';

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, {
                    type: 'DEPLOYMENT',
                    name,
                });
                cy.wait('@centralDeployment');

                cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);
                cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                    const nIngressOnly = $trs.has(ingressSelector).length;
                    const nEgressOnly = $trs.has(egressSelector).length;
                    const nBidirectional = $trs.has(bidirectionalSelector).length;
                    expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                    expect(nBidirectional).not.to.equal(0);

                    cy.get('#panel-close-button').click();

                    cy.getCytoscape(cytoscapeContainer).then((cytoscape2) => {
                        mouseOverNodeByName(cytoscape2, { type: 'DEPLOYMENT', name });

                        cy.get(selectors.tooltip.title).should('contain', name);
                        cy.get(selectors.tooltip.body)
                            .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                            .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                    });
                });
            });
        });

        it('has ingress only', () => {
            cy.intercept('GET', api.network.deployment, {
                fixture: 'network/scannerDbDeployment.json',
            }).as('scannerDbDeployment');
            visitNetworkGraphWithMockedData();

            const name = 'scanner-db';

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, {
                    type: 'DEPLOYMENT',
                    name,
                });
                cy.wait('@scannerDbDeployment');

                cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`).first();

                cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                    const nIngressOnly = $trs.has(ingressSelector).length;
                    const nEgressOnly = $trs.has(egressSelector).length;
                    const nBidirectional = $trs.has(bidirectionalSelector).length;
                    expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                    expect(nEgressOnly).to.equal(0);
                    expect(nBidirectional).to.equal(0);

                    cy.get('#panel-close-button').click();

                    cy.getCytoscape(cytoscapeContainer).then((cytoscape2) => {
                        mouseOverNodeByName(cytoscape2, { type: 'DEPLOYMENT', name });

                        cy.get(selectors.tooltip.title).should('contain', name);
                        cy.get(selectors.tooltip.body)
                            .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                            .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                    });
                });
            });
        });

        it('has egress only', () => {
            cy.intercept('GET', api.network.deployment, {
                fixture: 'network/collectorDeployment.json',
            }).as('collectorDeployment');
            visitNetworkGraphWithMockedData();

            const name = 'collector';

            cy.getCytoscape(networkPageSelectors.cytoscapeContainer).then((cytoscape) => {
                clickOnNodeByName(cytoscape, {
                    type: 'DEPLOYMENT',
                    name,
                });
                cy.wait('@collectorDeployment');

                cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);

                cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                    const nIngressOnly = $trs.has(ingressSelector).length;
                    const nEgressOnly = $trs.has(egressSelector).length;
                    const nBidirectional = $trs.has(bidirectionalSelector).length;
                    expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                    expect(nIngressOnly).to.equal(0);
                    expect(nBidirectional).to.equal(0);

                    cy.get('#panel-close-button').click();

                    cy.getCytoscape(cytoscapeContainer).then((cytoscape2) => {
                        mouseOverNodeByName(cytoscape2, { type: 'DEPLOYMENT', name });

                        cy.get(selectors.tooltip.title).should('contain', name);
                        cy.get(selectors.tooltip.body)
                            .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                            .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                    });
                });
            });
        });
    });
});
