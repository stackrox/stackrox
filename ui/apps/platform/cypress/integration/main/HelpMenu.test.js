import withAuth from '../../helpers/basicAuth';
import { visitMainDashboard } from '../../helpers/main';
import { interactAndWaitForResponses } from '../../helpers/request';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import { visit } from '../../helpers/visit';

const apiReferencePath = '/main/apidocs';

const apiReferenceAlias = 'docs/swagger';

const routeMatcherMapForReference = {
    [apiReferenceAlias]: '/api/docs/swagger',
};

const metadataAlias = 'metadata';

const routeMatcherMapForMetadata = {
    [metadataAlias]: '/v1/metadata',
};

describe('Help menu API Reference', () => {
    withAuth();

    const title = 'API Reference (v1)';

    it('should visit via menu on top nav', () => {
        visitMainDashboard();

        interactAndWaitForResponses(() => {
            cy.get('button[aria-label="Help menu"]').click();
            cy.get(`a:contains("${title}")`).click();
        }, routeMatcherMapForReference);

        cy.location('pathname').should('eq', apiReferencePath);
        cy.get(`h1:contains("${title}")`);
    });

    it('should visit via path', () => {
        visit(apiReferencePath, routeMatcherMapForReference);

        cy.get(`h1:contains("${title}")`);

        // Exception to pattern of separate test for title, because API Reference loads so slowly.
        cy.title().should('match', getRegExpForTitleWithBranding(title));
    });
});

describe('Help menu Help Center', () => {
    withAuth();

    it('should be a link', () => {
        visitMainDashboard();

        /*
         * Open Help menu, and then wait for an authenticated metadata request,
         * which is prerequisite to render the link below.
         */
        interactAndWaitForResponses(() => {
            cy.get('button[aria-label="Help menu"]').click();
        }, routeMatcherMapForMetadata);

        // Assert link, but do not click it (in case external web site is unavailable).
        cy.get('a:contains("Help Center")');
    });
});

describe('Help menu version number', () => {
    withAuth();

    it('should be a disabled link', () => {
        visitMainDashboard();

        /*
         * Open Help menu, and then wait for an authenticated metadata request,
         * which is prerequisite to render the version number below.
         */
        interactAndWaitForResponses(() => {
            cy.get('button[aria-label="Help menu"]').click();
        }, routeMatcherMapForMetadata);

        cy.get('nav[aria-label="Help menu"] a[role="menuitem"][aria-disabled="true"]');
    });
});
