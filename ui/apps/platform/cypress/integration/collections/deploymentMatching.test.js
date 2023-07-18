import withAuth from '../../helpers/basicAuth';
import {
    assertDeploymentsAreMatched,
    assertDeploymentsAreMatchedExactly,
    assertDeploymentsAreNotMatched,
    tryDeleteCollection,
    visitCollections,
} from './Collections.helpers';
import { collectionSelectors as selectors } from './Collections.selectors';

describe('Collection deployment matching', () => {
    withAuth();

    const sampleCollectionName = 'Stackrox sample deployments';
    const withEmbeddedCollectionName = 'Contains embedded collections';

    // Clean up when the test suite exits
    after(() => {
        tryDeleteCollection(withEmbeddedCollectionName);
        tryDeleteCollection(sampleCollectionName);
    });

    it('should preview deployments matching specified rules', () => {
        // Cleanup from potential previous test runs
        tryDeleteCollection(withEmbeddedCollectionName);
        tryDeleteCollection(sampleCollectionName);

        visitCollections();

        cy.get('a:contains("Create collection")').click();
        cy.get('input[name="name"]').type(sampleCollectionName);
        cy.get('input[name="description"]').type('Matches some stackrox deployments');

        cy.get('button:contains("All namespaces")').click();
        cy.get('button:contains("Namespaces with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the namespace name"]').type('stackrox');

        // Test that Stackrox deployments are matched
        assertDeploymentsAreMatched(['central', 'central-db', 'collector', 'scanner', 'sensor']);

        // Restrict collection to two specific deployments
        cy.get('button:contains("All deployments")').click();
        cy.get('button:contains("Deployments with labels matching")').click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 1 of 1"]').type(
            'app=collector'
        );
        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 2 of 2 for deployment rule 1 of 1"]').type(
            'app=sensor'
        );

        assertDeploymentsAreMatchedExactly(['collector', 'sensor']);

        cy.get('button:contains("Save")').click();

        cy.get(`td[data-label="Collection"] a:contains("${sampleCollectionName}")`);
    });

    // TODO Update criteria to work for both GKE and OpenShift.
    // This test relies on the creation of a collection in the previous test in order to check
    // the resolution of deployments with embedded collections.
    it.skip('should preview deployments using embedded collections', () => {
        // Cleanup from potential previous test runs
        tryDeleteCollection(withEmbeddedCollectionName);
        visitCollections();

        cy.get('a:contains("Create collection")').click();
        cy.get('input[name="name"]').type(withEmbeddedCollectionName);
        cy.get('input[name="description"]').type('Embeds another collection');

        cy.get('button:contains("All namespaces")').click();
        cy.get('button:contains("Namespaces with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the namespace name"]').type(
            'kube-system'
        );

        // Assert that results have loaded, but deployments beyond the first page are not visible
        assertDeploymentsAreMatched(['calico-node']);
        assertDeploymentsAreNotMatched(['kube-dns']);

        // target `kube-dns` deployment is on the second or third page of results, so load three pages
        // to ensure it is visible
        cy.get(selectors.viewMoreResultsButton).scrollIntoView();

        // load second page of results
        cy.get(selectors.viewMoreResultsButton).click();
        cy.get(`${selectors.viewMoreResultsButton}:not(.pf-m-in-progress)`).scrollIntoView();
        // load third page of results
        cy.get(selectors.viewMoreResultsButton).click();
        cy.get(`${selectors.viewMoreResultsButton}:not(.pf-m-in-progress)`).scrollIntoView();
        assertDeploymentsAreMatched(['kube-dns']);

        // Restrict collection to two specific deployments
        cy.get('button:contains("All deployments")').click();
        cy.get('button:contains("Deployments with labels matching")').click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 1 of 1"]').type(
            'k8s-app=calico-node-autoscaler'
        );

        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 2 of 2 for deployment rule 1 of 1"]').type(
            'k8s-app=kube-dns'
        );

        assertDeploymentsAreMatchedExactly(['kube-dns', 'calico-node-vertical-autoscaler']);

        // View another collection via modal
        cy.get(selectors.viewEmbeddedCollectionButton('Available', sampleCollectionName)).click();

        // Test that the results for only the other collection are visible in the modal results pane
        cy.get(`${selectors.modal} ${selectors.deploymentResults}`)
            .its('length')
            .should('be.eq', 2);
        cy.get(`${selectors.modal} ${selectors.deploymentResult('collector')}`);
        cy.get(`${selectors.modal} ${selectors.deploymentResult('sensor')}`);
        cy.get(selectors.modalClose).click();

        // Attach the collection, assert that embedded collection deployments are resolved
        cy.get(selectors.attachCollectionButton(sampleCollectionName)).click();
        assertDeploymentsAreMatchedExactly([
            'kube-dns',
            'calico-node-vertical-autoscaler',
            'collector',
            'sensor',
        ]);

        // Detach the collection, assert that embedded collection deployments are gone
        cy.get(selectors.detachCollectionButton(sampleCollectionName)).click();
        assertDeploymentsAreMatchedExactly(['kube-dns', 'calico-node-vertical-autoscaler']);

        // Re-attach and save
        cy.get(selectors.attachCollectionButton(sampleCollectionName)).click();
        cy.get('button:contains("Save")').click();

        cy.get(`td[data-label="Collection"] a:contains("${withEmbeddedCollectionName}")`);
    });

    // TODO Update criteria to work for both GKE and OpenShift.
    it.skip('should filter deployment results in the sidebar', () => {
        visitCollections();
        cy.get(`td[data-label="Collection"] a:contains("${withEmbeddedCollectionName}")`).click();

        // Filter to deployments with deployment name matching
        cy.get(selectors.resultsPanelFilterInput).type('c');
        cy.get(selectors.resultsPanelFilterSearch).click();

        assertDeploymentsAreMatchedExactly(['calico-node-vertical-autoscaler', 'collector']);

        // Filter to deployments in namespaces matching
        cy.get(selectors.resultsPanelFilterEntitySelect).click();
        cy.get(selectors.resultsPanelFilterEntitySelectOption('Namespace')).click();
        cy.get(selectors.resultsPanelFilterInput).type('stackrox');
        cy.get(selectors.resultsPanelFilterSearch).click();

        // Test that only stackrox deployments are visible
        assertDeploymentsAreMatchedExactly(['collector', 'sensor']);
    });
});
