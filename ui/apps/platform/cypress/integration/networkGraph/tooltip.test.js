import * as api from '../../constants/apiEndpoints';
import { selectors as networkPageSelectors } from '../../constants/NetworkPage';
import { url as riskURL, selectors as riskPageSelectors } from '../../constants/RiskPage';
import withAuth from '../../helpers/basicAuth';
import { mouseOverNodeByName } from '../../helpers/networkGraph';
import selectors from '../../selectors/index';

const { cytoscapeContainer } = networkPageSelectors;

describe('Network Graph tooltip', () => {
    withAuth();

    describe('deployment node', () => {
        beforeEach(() => {
            cy.server();
            cy.route('GET', api.risks.riskyDeployments).as('deployments');
        });

        const openSidePanelForDeployment = (name) => {
            cy.visit(riskURL);
            cy.wait('@deployments');

            cy.get(`${selectors.table.rows}:contains("${name}")`).first().click();
            cy.get(riskPageSelectors.viewDeploymentsInNetworkGraphButton).click();
        };

        const {
            table: { cells: cellsSelector },
        } = selectors;
        const ingressSelector = `${cellsSelector}:contains("ingress")`;
        const egressSelector = `${cellsSelector}:contains("egress")`;
        const bidirectionalSelector = `${cellsSelector}:contains("bidirectional")`;

        const getIngressFlowsText = (count) => `${count} ingress flows`;
        const getEgressFlowsText = (count) => `${count} egress flows`;

        // TODO: re-enable when https://stack-rox.atlassian.net/browse/ROX-5904 is fixed
        xit('has no bidirectional', () => {
            const name = 'sensor';
            openSidePanelForDeployment(name);

            cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);
            cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                const nIngressOnly = $trs.has(ingressSelector).length;
                const nEgressOnly = $trs.has(egressSelector).length;
                const nBidirectional = $trs.has(bidirectionalSelector).length;
                expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                expect(nBidirectional).to.equal(2);

                cy.get('#panel-close-button').click();

                cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
                    mouseOverNodeByName(cytoscape, { type: 'DEPLOYMENT', name });

                    cy.get(selectors.tooltip.title).should('contain', name);
                    cy.get(selectors.tooltip.body)
                        .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                        .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                });
            });
        });

        // TODO: re-enable when https://stack-rox.atlassian.net/browse/ROX-5904 is fixed
        xit('has bidirectional', () => {
            const name = 'central';
            openSidePanelForDeployment(name);

            cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);
            cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                const nIngressOnly = $trs.has(ingressSelector).length;
                const nEgressOnly = $trs.has(egressSelector).length;
                const nBidirectional = $trs.has(bidirectionalSelector).length;
                expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                expect(nBidirectional).not.to.equal(0);

                cy.get('#panel-close-button').click();

                cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
                    mouseOverNodeByName(cytoscape, { type: 'DEPLOYMENT', name });

                    cy.get(selectors.tooltip.title).should('contain', name);
                    cy.get(selectors.tooltip.body)
                        .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                        .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                });
            });
        });

        it('has ingress only', () => {
            const name = 'scanner-db';
            openSidePanelForDeployment(name);

            cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`).first();
            cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                const nIngressOnly = $trs.has(ingressSelector).length;
                const nEgressOnly = $trs.has(egressSelector).length;
                const nBidirectional = $trs.has(bidirectionalSelector).length;
                expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                expect(nEgressOnly).to.equal(0);
                expect(nBidirectional).to.equal(0);

                cy.get('#panel-close-button').click();

                cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
                    mouseOverNodeByName(cytoscape, { type: 'DEPLOYMENT', name });

                    cy.get(selectors.tooltip.title).should('contain', name);
                    cy.get(selectors.tooltip.body)
                        .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                        .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                });
            });
        });

        it('has egress only', () => {
            const name = 'collector';
            openSidePanelForDeployment(name);

            cy.get(`${networkPageSelectors.detailsPanel.header}:contains("${name}")`);
            cy.get(networkPageSelectors.detailsPanel.table.rows).then(($trs) => {
                const nIngressOnly = $trs.has(ingressSelector).length;
                const nEgressOnly = $trs.has(egressSelector).length;
                const nBidirectional = $trs.has(bidirectionalSelector).length;
                expect(nIngressOnly + nEgressOnly + nBidirectional).to.equal($trs.length);

                expect(nIngressOnly).to.equal(0);
                expect(nBidirectional).to.equal(0);

                cy.get('#panel-close-button').click();

                cy.getCytoscape(cytoscapeContainer).then((cytoscape) => {
                    mouseOverNodeByName(cytoscape, { type: 'DEPLOYMENT', name });

                    cy.get(selectors.tooltip.title).should('contain', name);
                    cy.get(selectors.tooltip.body)
                        .should('contain', getIngressFlowsText(nIngressOnly + nBidirectional))
                        .should('contain', getEgressFlowsText(nEgressOnly + nBidirectional));
                });
            });
        });
    });
});
