import { selectors as networkPageSelectors } from '../constants/NetworkPage';
import selectors from '../selectors/index';
import toastSelectors from '../selectors/toast';
import navigationSelectors from '../selectors/navigation';

import * as api from '../constants/apiEndpoints';
import withAuth from '../helpers/basicAuth';
import {
    viewRiskDeploymentByName,
    viewRiskDeploymentInNetworkGraph,
    visitRiskDeployments,
} from '../helpers/risk';
import {
    visitNetworkGraph,
    visitNetworkGraphFromLeftNav,
    visitNetworkGraphWithMockedData,
    visitNetworkGraphWithNamespaceFilter,
} from '../helpers/networkGraph';

function uploadYAMLFile(fileName, selector) {
    cy.intercept('POST', api.network.simulate).as('postNetworkPolicySimulate');

    // Needs force option because input element has display: none style.
    cy.get(selector).selectFile(`${Cypress.config('fixturesFolder')}/${fileName}`, { force: true });

    cy.wait('@postNetworkPolicySimulate');
}

describe('Network page', () => {
    withAuth();

    it('should visit using the left nav', () => {
        visitNetworkGraphFromLeftNav();
        cy.get('h1:contains("Network Graph")');
    });

    it('should have selected item in nav bar', () => {
        visitNetworkGraph();
        cy.get(`${navigationSelectors.navLinks}:contains('Network')`).should(
            'have.class',
            'pf-m-current'
        );
    });

    it('should display a legend', () => {
        visitNetworkGraphWithMockedData();

        const { deployments, namespaces, connections } = networkPageSelectors.legend;

        cy.get(`${deployments} *:nth-child(1) [alt="deployment"]`);
        cy.get(`${deployments} *:nth-child(2) [alt="deployment-external-connections"]`);
        cy.get(`${deployments} *:nth-child(3) [alt="deployment-allowed-connections"]`);
        cy.get(`${deployments} *:nth-child(4) [alt="non-isolated-deployment-allowed"]`);

        cy.get(`${namespaces} *:nth-child(1) [alt="namespace"]`);
        cy.get(`${namespaces} *:nth-child(2) [alt="namespace-allowed-connection"]`);
        cy.get(`${namespaces} *:nth-child(3) [alt="namespace-connection"]`);

        cy.get(`${connections} *:nth-child(1) [alt="active-connection"]`);
        cy.get(`${connections} *:nth-child(2) [alt="allowed-connection"]`);
        cy.get(`${connections} *:nth-child(3) [alt="namespace-egress-ingress"]`);
    });

    it('should handle toggle click on simulator network policy button', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        cy.get(networkPageSelectors.buttons.viewActiveYamlButton).should('be.visible');
        cy.get(networkPageSelectors.panels.creatorPanel).should('be.visible');
        cy.get(networkPageSelectors.buttons.stopSimulation).click();
        cy.get(networkPageSelectors.panels.creatorPanel).should('not.exist');
    });

    it('should display expected toast message when uploaded yaml without namespace', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadYAMLFile('network/policywithoutnamespace.yaml', 'input[type="file"]');

        cy.get(networkPageSelectors.simulatorSuccessMessage);
        cy.get(networkPageSelectors.buttons.applyNetworkPolicies).click();
        cy.get(networkPageSelectors.buttons.apply).click();
        cy.get(`${toastSelectors.body}:contains("network policy has empty namespace")`);
    });

    it('should display display policies processed message when uploaded yaml with namespace', () => {
        visitNetworkGraphWithMockedData();

        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();
        uploadYAMLFile('network/policywithnamespace.yaml', 'input[type="file"]');

        cy.get(networkPageSelectors.simulatorSuccessMessage);
        // Stop here because after Policies processed, local deployment differs from CI.
    });

    it('should show the network policy simulator screen after generating network policies', () => {
        visitRiskDeployments();
        viewRiskDeploymentByName('central');
        viewRiskDeploymentInNetworkGraph();

        cy.get(networkPageSelectors.networkEntityTabbedOverlay.header).should('be.visible');
        cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();

        cy.intercept('GET', api.network.generate).as('getNetworkPolicyGenerate');
        cy.intercept('POST', api.network.simulate).as('getNetworkPolicySimulate');
        cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
        cy.wait(['@getNetworkPolicyGenerate', '@getNetworkPolicySimulate']);

        cy.get(networkPageSelectors.panels.simulatorPanel).should('be.visible');
    });
});

describe('Network Deployment Details', () => {
    withAuth();

    it('should show the deployment name and namespace', () => {
        visitRiskDeployments();
        viewRiskDeploymentByName('central');
        viewRiskDeploymentInNetworkGraph();

        cy.get(`${selectors.tab.tabs}:contains('Details')`).click();
        cy.get(`[data-testid="Deployment Name"]:contains('central')`);
        cy.get(`[data-testid="Namespace"]:contains('stackrox')`);
    });
});

describe('Network Policy Simulator', () => {
    withAuth();

    it('should update the graph when generating and simulating network policies', () => {
        // this will get the deployments for the 'default' and 'docker' namespace
        function getDeployments(cytoscape) {
            const deployments = cytoscape.filter((element) => {
                return (
                    element.isNode() &&
                    element.data('type') === 'DEPLOYMENT' &&
                    (element.data('parent') === 'default' || element.data('parent') === 'docker')
                );
            });
            return deployments;
        }

        visitNetworkGraphWithNamespaceFilter('stackrox');

        cy.get(networkPageSelectors.buttons.allowedFilter).click();
        cy.getCytoscape('#cytoscapeContainer').then((cytoscape) => {
            const deployments = getDeployments(cytoscape);
            // we want to make sure all the deployments from 'default' and 'docker' namespaces are non-isolated
            deployments.forEach((deployment) => {
                expect(deployment.hasClass('nonIsolated')).to.equal(true);
            });
            cy.get(networkPageSelectors.buttons.simulatorButtonOff).click();

            cy.intercept('GET', api.network.generate).as('getNetworkPolicyGenerate');
            cy.intercept('POST', api.network.simulate).as('getNetworkPolicySimulate');
            cy.get(networkPageSelectors.buttons.generateNetworkPolicies).click();
            cy.wait(['@getNetworkPolicyGenerate', '@getNetworkPolicySimulate']);

            cy.getCytoscape('#cytoscapeContainer').then((updatedCytoscape) => {
                const simulatedDeployments = getDeployments(updatedCytoscape);
                // After the simulated graph, we want to make sure all the deployments from 'default' and 'docker' namespaces are not non-isolated
                simulatedDeployments.forEach((deployment) => {
                    expect(deployment.hasClass('nonIsolated')).to.equal(false);
                });
            });
        });
    });
});

describe('Network Flows Table', () => {
    withAuth();

    it('should show the proper table column headers for the network flows table', () => {
        visitRiskDeployments();
        viewRiskDeploymentByName('central');
        viewRiskDeploymentInNetworkGraph();

        cy.get(`${selectors.tab.tabs}:contains('Network Flows')`).click();
        cy.get(`${selectors.table.th}:contains('Entity')`);
        cy.get(`${selectors.table.th}:contains('Traffic')`);
        cy.get(`${selectors.table.th}:contains('Type')`);
        cy.get(`${selectors.table.th}:contains('Namespace')`);
        cy.get(`${selectors.table.th}:contains('State')`);

        cy.get(`${selectors.table.th}:contains('Protocol')`);
        cy.get(`${selectors.table.th}:contains('Port')`);
    });
});
