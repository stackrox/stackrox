import withAuth from '../../helpers/basicAuth';
import {
    assertCannotFindThePage,
    visit,
    visitWithStaticResponseForPermissions,
} from '../../helpers/visit';
import navSelectors from '../../selectors/navigation';
import { tryCreateCollection, tryDeleteCollection } from './Collections.helpers';
import { collectionSelectors } from './Collections.selectors';

describe('Collection permission checks', () => {
    withAuth();

    const collectionName = 'Permission test collection';

    before(() => {
        const rules = [{ fieldName: 'Namespace', values: [{ value: 'stackrox' }], operator: 'OR' }];

        tryCreateCollection(collectionName, 'e2e test description', [], [{ rules }]);
    });

    after(() => tryDeleteCollection(collectionName));

    it('should prevent users with no access from viewing collections', () => {
        cy.fixture('auth/mypermissionsMinimalAccess.json').then(({ resourceToAccess }) => {
            const staticResponseForPermissions = {
                body: {
                    resourceToAccess: { ...resourceToAccess, WorkflowAdministration: 'NO_ACCESS' },
                },
            };

            // Mock a 'NO_ACCESS' permission response
            visitWithStaticResponseForPermissions('/main', staticResponseForPermissions);

            // Expand the Platform Config section for ease of debugging
            cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`).click();
            cy.get(`${navSelectors.nestedNavLinks}:contains("Collections")`).should('not.exist');

            // Test direct visit via URL
            visit('/main/collections');
            // The Collections header should not be present, and a not found 404 message will be displayed
            cy.get('h1:contains("Collections")').should('not.exist');
            assertCannotFindThePage();
        });
    });

    it('should not provide mutable UI controls to users with read-only access', () => {
        // Mock a 'READ_ACCESS' permission response
        visitWithStaticResponseForPermissions('/main', {
            body: { resourceToAccess: { WorkflowAdministration: 'READ_ACCESS' } },
        });
        // Ensure the collections link is visible and takes the user to the collections table
        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`).click();
        cy.get(`${navSelectors.nestedNavLinks}:contains("Collections")`).click();

        cy.get('h1:contains("Collections")');
        // Ensure the 'Create collection' button does not exist
        cy.get('*:contains("Create collection")').should('not.exist');

        const linkSelector = collectionSelectors.tableLinkByName(collectionName);
        // Check existence of row before negative assertion
        cy.get(`tr:has(${linkSelector})`);
        // Ensure table rows do not have an action menu
        cy.get(`tr:has(${linkSelector}) button[aria-label="Actions"]`).should('not.exist');
        // Visit page for individual collection and verify action button is not present
        cy.get(linkSelector).click();
        cy.get(`h1:contains("${collectionName}")`);
        cy.get(`button:contains("Actions")`).should('not.exist');
        cy.get(`button:contains("Save")`).should('not.exist');
    });

    it('should provide the full UI to users with read-write access', () => {
        // Mock a 'READ_WRITE_ACCESS' permission response
        visitWithStaticResponseForPermissions('/main', {
            body: { resourceToAccess: { WorkflowAdministration: 'READ_WRITE_ACCESS' } },
        });
        // Ensure the collections link is visible and takes the user to the collections table
        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`).click();
        cy.get(`${navSelectors.nestedNavLinks}:contains("Collections")`).click();

        cy.get('h1:contains("Collections")');
        // Ensure the 'Create collection' button is visible
        cy.get('*:contains("Create collection")');

        const linkSelector = collectionSelectors.tableLinkByName(collectionName);
        // Ensure that menu options in table rows are available
        cy.get(`tr:has(${linkSelector}) button[aria-label="Actions"]`).click();
        cy.get(`tr:has(${linkSelector}) button:contains("Edit collection")`);
        cy.get(`tr:has(${linkSelector}) button:contains("Clone collection")`);
        cy.get(`tr:has(${linkSelector}) button:contains("Delete collection")`);

        // Visit page for individual collection and verify action button is present
        cy.get(linkSelector).click();
        cy.get(`h1:contains("${collectionName}")`);
        cy.get(`button:contains("Actions")`).click();
        cy.get(`ul[role="menu"] button:contains("Edit collection")`);
        cy.get(`ul[role="menu"] button:contains("Clone collection")`);
        cy.get(`ul[role="menu"] button:contains("Delete collection")`);

        // Enter edit mode and verify "save" button is present
        cy.get(`ul[role="menu"] button:contains("Edit collection")`).click();
        cy.get(`button:contains("Save")`);
    });
});
