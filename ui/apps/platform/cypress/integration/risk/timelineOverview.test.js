import { selectors, url } from '../../constants/RiskPage';

import * as api from '../../constants/apiEndpoints';

import withAuth from '../../helpers/basicAuth';

function setRoutes() {
    cy.server();
    cy.route('GET', api.risks.riskyDeployments).as('deployments');
    cy.route('GET', api.risks.fetchDeploymentWithRisk).as('getDeployment');
}

function openDeployment(deploymentName) {
    cy.visit(url);
    cy.wait('@deployments');

    cy.get(`${selectors.table.rows}:contains(${deploymentName})`).click();
    cy.wait('@getDeployment');
}

describe('Risk Page Event Timeline - Timeline Overview', () => {
    withAuth();

    it('should show the timeline graph when the overview is clicked', () => {
        setRoutes();
        // select a deployment to open the side panel
        openDeployment('collector');
        // open the process discovery tab
        cy.get(selectors.sidePanel.processDiscoveryTab).click();
        // click the overview button
        cy.get(selectors.eventTimelineOverviewButton).click();
        // the event timeline graph should show up
        cy.get(selectors.eventTimeline.timeline);
    });
});
