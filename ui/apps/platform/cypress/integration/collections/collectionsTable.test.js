import withAuth from '../../helpers/basicAuth';
import { visitCollections, visitCollectionsFromLeftNav } from '../../helpers/collections';
import { hasFeatureFlag } from '../../helpers/features';

describe('Collections table', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_OBJECT_COLLECTIONS')) {
            this.skip();
        }
    });

    it('should visit via link in left nav', () => {
        visitCollectionsFromLeftNav();
    });

    it('should visit via page address', () => {
        visitCollections();
    });

    it('should have table column headings', () => {
        visitCollections();

        cy.get('th:contains("Collection")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("In use")');
    });

    it('should have button to create collection if role has READ_WRITE_ACCESS', () => {
        visitCollections();

        cy.get('button:contains("Create collection")');
    });
});
