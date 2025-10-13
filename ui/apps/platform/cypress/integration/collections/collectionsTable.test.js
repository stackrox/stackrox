import withAuth from '../../helpers/basicAuth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import {
    collectionsAlias,
    collectionsCountAlias,
    visitCollections,
    visitCollectionsFromLeftNav,
} from './Collections.helpers';

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

    it('should visit via link in left nav', () => {
        visitCollectionsFromLeftNav(staticResponseMap);
    });

    it('should visit via page address', () => {
        visitCollections(staticResponseMap);
    });

    it('should have plural title and table column headings', () => {
        visitCollections(staticResponseMap);

        cy.title().should('match', getRegExpForTitleWithBranding('Collections'));

        cy.get('th:contains("Collection")');
        cy.get('th:contains("Description")');
    });

    it('should have button to create collection if role has READ_WRITE_ACCESS', () => {
        visitCollections(staticResponseMap);

        cy.get('a:contains("Create collection")');
    });
});
