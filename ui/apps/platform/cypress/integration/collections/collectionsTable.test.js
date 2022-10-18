import withAuth from '../../helpers/basicAuth';
import {
    collectionsAlias,
    collectionsCountAlias,
    visitCollections,
    visitCollectionsFromLeftNav,
} from '../../helpers/collections';
import { hasFeatureFlag } from '../../helpers/features';

// Mock responses until endpoints are implemented.

const collections = [];
const count = collections.length;

const staticResponseMap = {
    [collectionsAlias]: {
        body: { collections },
    },
    [collectionsCountAlias]: {
        body: { count },
    },
};

describe('Collections table', () => {
    withAuth();

    before(function beforeHook() {
        if (!hasFeatureFlag('ROX_OBJECT_COLLECTIONS')) {
            this.skip();
        }
    });

    it('should visit via link in left nav', () => {
        visitCollectionsFromLeftNav(staticResponseMap);
    });

    it('should visit via page address', () => {
        visitCollections(staticResponseMap);
    });

    it('should have table column headings', () => {
        visitCollections(staticResponseMap);

        cy.get('th:contains("Collection")');
        cy.get('th:contains("Description")');
        cy.get('th:contains("In use")');
    });

    it('should have button to create collection if role has READ_WRITE_ACCESS', () => {
        visitCollections(staticResponseMap);

        cy.get('a:contains("Create collection")');
    });
});
