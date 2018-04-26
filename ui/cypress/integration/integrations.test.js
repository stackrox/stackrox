import { selectors } from './pages/IntegrationsPage';
import * as api from './apiEndpoints';

describe('Integrations page', () => {
    beforeEach(() => {
        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click();
    });

    it('should have selected item in nav bar', () => {
        cy.get(selectors.configure).should('have.class', 'bg-primary-600');
    });

    it('should allow integration with Slack', () => {
        cy.get('div.ReactModalPortal').should('not.exist');

        cy.get('button:contains("Slack")').click();
        cy.get('div.ReactModalPortal');
    });
});

describe('Cluster Creation Flow', () => {
    beforeEach(() => {
        cy.server();
        cy.fixture('clusters/single.json').as('singleCluster');
        cy.route('GET', api.clusters.list, '@singleCluster').as('clusters');
        cy.route('POST', api.clusters.zip, {}).as('download');
        cy.route('POST', api.clusters.list, { id: 'kubeCluster1' }).as('addCluster');
        cy.visit('/');
        cy.get(selectors.configure).click();
        cy.get(selectors.navLink).click();
        cy.wait('@clusters');
    });

    it('Should show the remote cluster when clicking the Docker Swarm tile', () => {
        cy.get(selectors.dockerSwarmTile).click();

        cy.get(selectors.clusters.swarmCluster1);
    });

    it('Should show a disabled form when viewing a specific cluster', () => {
        cy.get(selectors.dockerSwarmTile).click();

        cy.get(selectors.clusters.swarmCluster1).click();

        cy
            .get(selectors.readOnlyView)
            .eq(0)
            .should('have.text', 'Name:Swarm Cluster 1');

        cy
            .get(selectors.readOnlyView)
            .eq(1)
            .should('have.text', 'Cluster Type:SWARM_CLUSTER');

        cy
            .get(selectors.readOnlyView)
            .eq(2)
            .should('have.text', 'Image name (Prevent location):stackrox/prevent:latest');

        cy
            .get(selectors.readOnlyView)
            .eq(3)
            .should('have.text', 'Central API Endpoint:central.stackrox:443');
    });

    it('Should be able to view a form with the necessary cluster entity fields when clicking "Add"', () => {
        cy.get(selectors.dockerSwarmTile).click();

        cy.get(selectors.buttons.addCluster).click();

        cy
            .get(selectors.form.cluster.inputs)
            .eq(0)
            .type('Kubernetes Cluster 1');
        cy
            .get(selectors.form.cluster.inputs)
            .eq(1)
            .type('KUBERNETES_CLUSTER');
        cy
            .get(selectors.form.cluster.inputs)
            .eq(2)
            .type('stackrox/prevent:latest');
        cy
            .get(selectors.form.cluster.inputs)
            .eq(3)
            .type('central.prevent_net:443');
        cy
            .get(selectors.form.cluster.inputs)
            .eq(4)
            .type('stackrox');
        cy.get(selectors.form.cluster.checkbox).check();

        cy.get(selectors.buttons.next).click();
        cy.wait('@addCluster');

        cy.get(selectors.buttons.download, { timeout: 500 }).click();
        cy.wait('@download');
    });
});
