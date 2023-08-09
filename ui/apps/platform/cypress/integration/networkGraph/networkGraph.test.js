import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';

import {
    visitNetworkGraph,
    visitNetworkGraphFromLeftNav,
    checkNetworkGraphEmptyState,
    selectCluster,
    selectNamespace,
    selectDeployment,
    selectFilter,
    updateAndCloseCidrModal,
} from './networkGraph.helpers';
import { networkGraphSelectors } from './networkGraph.selectors';

describe('Network Graph smoke tests', () => {
    withAuth();

    it('should visit using the left nav', () => {
        visitNetworkGraphFromLeftNav();

        checkNetworkGraphEmptyState();
    });

    it('should visit from direct navigation', () => {
        visitNetworkGraph();

        cy.title().should('match', getRegExpForTitleWithBranding('Network Graph'));

        checkNetworkGraphEmptyState();
    });

    it('should render a graph, including toolbar, when cluster and namespace are selected', () => {
        visitNetworkGraph();

        checkNetworkGraphEmptyState();

        selectCluster();
        selectNamespace('stackrox');

        // check that group of nodes for NS is present
        cy.get(`${networkGraphSelectors.groups} [data-id="stackrox"]`);

        // check that label for NS is present and has the filtered-namespace class
        cy.get(
            `${networkGraphSelectors.nodes} [data-id="stackrox"] g.filtered-namespace text`
        ).contains('stackrox');

        // check that toolbar and buttons are present
        cy.get(`${networkGraphSelectors.toolbar}`);
        cy.get(networkGraphSelectors.toolbarItem).contains('Zoom In');
        cy.get(networkGraphSelectors.toolbarItem).contains('Zoom Out');
        cy.get(networkGraphSelectors.toolbarItem).contains('Fit to Screen');
        cy.get(networkGraphSelectors.toolbarItem).contains('Reset View');

        // open Legend as well, after verifying its existence
        cy.get(networkGraphSelectors.toolbarItem).contains('Legend').click();

        // check Legend content
        cy.get('.pf-c-popover__content [data-testid="legend-title"]:contains("Legend")');

        cy.get('.pf-c-popover__content [data-testid="node-types-title"]:contains("Node types")');
        cy.get('.pf-c-popover__content .pf-c-description-list__text:contains("Deployment")');
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("External CIDR block")'
        );

        cy.get(
            '.pf-c-popover__content [data-testid="namespace-types-title"]:contains("Namespace types")'
        );
        cy.get('.pf-c-popover__content .pf-c-description-list__text:contains("Related namespace")');
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Filtered namespace")'
        );

        cy.get(
            '.pf-c-popover__content [data-testid="deployment-badges-title"]:contains("Deployment badges")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Connected to external entities")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Isolated by network policy rules")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("All traffic allowed (No network policies)")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Only has an egress network policy")'
        );
        cy.get(
            '.pf-c-popover__content .pf-c-description-list__text:contains("Only has an ingress network policy")'
        );

        // close the Legend
        cy.get('.pf-c-popover__content [aria-label="Close"]').click();
    });

    it('should correctly display entities when scope and filters are applied', () => {
        visitNetworkGraph();

        // Apply a namespace filter for 'stackrox'
        selectNamespace('stackrox');

        // Verify that 'stackrox' and 'kube-system' namespaces are present
        cy.get(networkGraphSelectors.filteredNamespaceGroupNode('stackrox'));
        cy.get(networkGraphSelectors.relatedNamespaceGroupNode('kube-system'));

        // Verify that central, central-db, scanner, scanner-db, sensor are present
        ['central', 'central-db', 'scanner', 'scanner-db', 'sensor'].forEach((deployment) => {
            cy.get(networkGraphSelectors.deploymentNode(deployment));
        });

        // Apply a deployment filter for 'central-db'
        selectDeployment('central-db');

        // Verify that central, central-db are present and that scanner, sensor, sensor-db are not present
        ['central', 'central-db'].forEach((deployment) => {
            cy.get(networkGraphSelectors.deploymentNode(deployment));
        });
        ['scanner', 'scanner-db', 'sensor'].forEach((deployment) => {
            cy.get(networkGraphSelectors.deploymentNode(deployment)).should('not.exist');
        });

        // Remove the central-db selection from the scope filter
        selectDeployment('central-db');
        // Apply a general filter of "Deployment Label" for 'app=central-db'
        selectFilter('Deployment Label', 'app=central-db');

        ['central', 'central-db'].forEach((deployment) => {
            cy.get(networkGraphSelectors.deploymentNode(deployment));
        });
        ['scanner', 'scanner-db', 'sensor'].forEach((deployment) => {
            cy.get(networkGraphSelectors.deploymentNode(deployment)).should('not.exist');
        });

        // Verify that the correct namespaces are displayed
        cy.get(networkGraphSelectors.filteredNamespaceGroupNode('stackrox'));
        cy.get(networkGraphSelectors.relatedNamespaceGroupNode('kube-system')).should('not.exist');
    });

    it('should allow the addition and deletion of CIDR blocks', () => {
        visitNetworkGraph();

        // open the CIDR block modal and add a block
        cy.get(networkGraphSelectors.manageCidrBlocksButton).click();
        cy.get(networkGraphSelectors.cidrBlockEntryNameInputAt(0)).type('{selectall}redhat.com');
        cy.get(networkGraphSelectors.cidrBlockEntryCidrInputAt(0)).type('{selectall}10.0.0.0/24');

        updateAndCloseCidrModal();

        // Check that the values are still there
        cy.get(networkGraphSelectors.manageCidrBlocksButton).click();
        cy.get(networkGraphSelectors.cidrBlockEntryNameInputAt(0)).should(
            'have.value',
            'redhat.com'
        );
        cy.get(networkGraphSelectors.cidrBlockEntryCidrInputAt(0)).should(
            'have.value',
            '10.0.0.0/24'
        );
        cy.get(networkGraphSelectors.cidrBlockEntryNameInputAt(1)).should('not.exist');

        // Delete the CIDR block
        cy.get(networkGraphSelectors.cidrBlockEntryDeleteButtonAt(0)).click();

        updateAndCloseCidrModal();

        // Check that the values are removed
        cy.get(networkGraphSelectors.manageCidrBlocksButton).click();
        cy.get(networkGraphSelectors.cidrBlockEntryNameInputAt(0)).should('not.exist');
    });
});
