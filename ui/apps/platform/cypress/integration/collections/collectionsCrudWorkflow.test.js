import withAuth from '../../helpers/basicAuth';
import { visitCollections } from '../../helpers/collections';
import { hasFeatureFlag } from '../../helpers/features';

const baseUrl = '/v1/collections';
const autocompleteUrl = `${baseUrl}/autocomplete`;

// Cleanup an existing collection via API call
function tryDeleteCollection(collectionName) {
    const auth = { bearer: Cypress.env('ROX_AUTH_TOKEN') };

    cy.request({
        url: `${baseUrl}?query.query=Collection Name:"${collectionName}"`,
        auth,
    }).as('listCollections');

    cy.get('@listCollections').then((res) => {
        const collection = res.body.collections.find(({ name }) => name === collectionName);
        if (collection) {
            const { id } = collection;
            const url = `${baseUrl}/${id}`;
            cy.request({ url, auth, method: 'DELETE' });
        }
    });
}

/* 
    Each test in this spec builds upon the previous by executing another piece
    of the collection CRUD workflow.
*/
describe('Create collection', () => {
    withAuth();

    beforeEach(function beforeHook() {
        if (!hasFeatureFlag('ROX_OBJECT_COLLECTIONS')) {
            this.skip();
        }
        // Ignore autocomplete requests
        // TODO Remove this once the feature is in
        cy.intercept(autocompleteUrl, {});
    });

    const collectionName = 'Financial deployments';
    const clonedName = `${collectionName} (COPY)`;

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
        cy.get('input[aria-label="Select label key for deployment rule 1 of 1"]').type('meta/name');
        cy.get(`button:contains('Add "meta/name"')`).click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 1 of 1"]').type(
            'visa.*'
        );
        cy.get(`button:contains('Add "visa.*"')`).click();
        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 2 of 2 for deployment rule 1 of 1"]').type(
            'mastercard.*'
        );
        cy.get(`button:contains('Add "mastercard.*"')`).click();

        cy.get('button:contains("All namespaces")').click();
        cy.get('button:contains("Namespaces with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the namespace name"]').type('payments');
        cy.get(`button:contains('Add "payments"')`).click();

        cy.get('button:contains("All clusters")').click();
        cy.get('button:contains("Clusters with names matching")').click();
        cy.get('input[aria-label="Select value 1 of 1 for the cluster name"]').type('production');
        cy.get(`button:contains('Add "production"')`).click();

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
        cy.get('input[aria-label="Select label key for deployment rule 2 of 2"]').type(
            'meta/net-visibility'
        );
        cy.get(`button:contains('Add "meta/net-visibility"')`).click();
        cy.get('input[aria-label="Select label value 1 of 1 for deployment rule 2 of 2"]').type(
            'public-facing'
        );
        cy.get(`button:contains('Add "public-facing"')`).click();

        cy.get('button[aria-label="Add deployment label value for rule 1"]').click();
        cy.get('input[aria-label="Select label value 3 of 3 for deployment rule 1 of 2"]').type(
            'discover.*'
        );
        cy.get(`button:contains('Add "discover.*"')`).click();

        cy.get(`button[aria-label='Delete mastercard.*']`).click();

        cy.get('button[aria-label="Add cluster name value"]').click();
        cy.get('input[aria-label="Select value 2 of 2 for the cluster name"]').type('staging');
        cy.get(`button:contains('Add "staging"')`).click();

        cy.get('input[aria-label="Select value 1 of 2 for the cluster name"]').type(
            '{selectAll}security'
        );
        cy.get(`button:contains('Add "security"')`).click();

        // Save
        cy.get('button:contains("Save")').click();

        // Revisit the collection page and verify that the changes have stuck
        cy.get('a:contains("Financial deployments")').click();

        // Check "byLabel" inputs for deployment
        cy.get(`input[aria-label^="Select label value"][value="visa.*"]`);
        cy.get(`input[aria-label^="Select label value"][value="discover.*"]`);
        cy.get(`input[aria-label^="Select label value"][value="mastercard.*"]`).should('not.exist');

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
