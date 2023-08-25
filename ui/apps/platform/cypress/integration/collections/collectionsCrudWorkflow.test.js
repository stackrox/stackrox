import withAuth from '../../helpers/basicAuth';
import { tryDeleteCollection, visitCollections } from './Collections.helpers';

/* 
    Each test in this spec builds upon the previous by executing another piece
    of the collection CRUD workflow.
*/
describe('Create collection', () => {
    withAuth();

    const collectionName = 'Financial deployments';
    const clonedName = `${collectionName} -COPY-`;

    it('should allow creation of a new collection', () => {
        // Cleanup from potential previous test runs
        tryDeleteCollection(collectionName);
        tryDeleteCollection(clonedName);

        visitCollections();

        cy.get('a:contains("Create collection")').click();
        cy.get('input[name="name"]').type(collectionName);
        cy.get('input[name="description"]').type('A collection for financial data');

        cy.get('button:contains("All deployments")').click();
        cy.get('button:contains("Deployments with labels matching")').click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 1 of 1"]').type(
            'meta/name=visa-processor'
        );
        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 2 of 2 for deployment rule 1 of 1"]').type(
            'meta/name=mastercard-processor'
        );

        cy.get('button:contains("All namespaces")').click();
        cy.get('button:contains("Namespaces with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the namespace name"]').type('payments');

        cy.get('button:contains("All clusters")').click();
        cy.get('button:contains("Clusters with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the cluster name"]').type('production');

        cy.get('button:contains("Save")').click();

        cy.get(`td[data-label="Collection"] a:contains("${collectionName}")`);
    });

    it('should allow editing an existing collection', () => {
        visitCollections();

        // Make changes to an existing collection
        cy.get(`a:contains("${collectionName}")`).click();
        cy.get(`button:contains("Actions")`).click();
        cy.get(`button:contains("Edit collection")`).click();

        cy.get('button[aria-label="Add deployment label rule"]').click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 2 of 2"]').type(
            'meta/net-visibility=public-facing'
        );

        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 3 of 3 for deployment rule 1 of 2"]').type(
            'meta/name=discover-processor'
        );

        cy.get(`button[aria-label='Delete meta/name=mastercard-processor']`).click();

        cy.get('button[aria-label="Add cluster name value"]').click();
        cy.get('input[aria-label="Select value 2 of 2 for the cluster name"]').type('staging');

        cy.get('input[aria-label="Select value 1 of 2 for the cluster name"]').type(
            '{selectAll}security'
        );

        // Save
        cy.get('button:contains("Save")').click();

        // Revisit the collection page and verify that the changes have stuck
        cy.get('a:contains("Financial deployments")').click();

        // Check "byLabel" inputs for deployment
        cy.get(`input[aria-label^="Select label value"][value="meta/name=visa-processor"]`);
        cy.get(`input[aria-label^="Select label value"][value="meta/name=discover-processor"]`);
        cy.get(
            `input[aria-label^="Select label value"][value="meta/name=mastercard-processor"]`
        ).should('not.exist');

        // Check "byName" inputs for namespace
        cy.get(`input[aria-label$="for the namespace name"][value="payments"]`);

        // Check "byName" inputs for cluster
        cy.get(`input[aria-label$="for the cluster name"][value="staging"]`);
        cy.get(`input[aria-label$="for the cluster name"][value="security"]`);
        cy.get(`input[aria-label$="for the cluster name"][value="production"]`).should('not.exist');
    });

    it('should allow cloning an existing collection', () => {
        // Cleanup from potential previous test runs
        tryDeleteCollection(clonedName);
        visitCollections();

        // Make changes to an existing collection
        cy.get(`a:contains("${collectionName}")`).click();
        cy.get(`button:contains("Actions")`).click();
        cy.get(`button:contains("Clone collection")`).click();

        // Clone it with the default values
        cy.get(`input[name="name"][value="${clonedName}"]`);
        cy.get('button:contains("Save")').click();

        // Ensure both collections are available
        cy.get(`td[data-label="Collection"] a:contains("${collectionName}")`);
        cy.get(`td[data-label="Collection"] a:contains("${clonedName}")`);
    });

    it('should delete the previously created collections', () => {
        // From the main table
        visitCollections();

        // Delete the clone first, since the `:contains()` selector cannot do an exact match
        // Delete one from the main collection table
        cy.get(`tr:has(a:contains("${clonedName}")) button[aria-label="Actions"]`).click();
        cy.get('button:contains("Delete collection")').click();
        cy.get('*[role="dialog"] button:contains("Delete")').click();
        cy.get('*:contains("Successfully deleted")');
        cy.get(`td[data-label="Collection"] a:contains("${clonedName}")`).should('not.exist');

        // Delete one from the individual collection page
        cy.get(`td[data-label="Collection"] a:contains("${collectionName}")`).click();
        cy.get(`button:contains("Actions")`).click();
        cy.get(`button:contains("Delete collection")`).click();
        cy.get('*[role="dialog"] button:contains("Delete")').click();

        // Should navigate back to the main collections table
        cy.get('h1:contains("Collections")');

        // Both collections should be gone from the main table
        cy.get(`td[data-label="Collection"] a:contains("${collectionName}")`).should('not.exist');
    });
});
