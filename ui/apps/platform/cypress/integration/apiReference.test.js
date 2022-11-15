import withAuth from '../helpers/basicAuth';
import { visitMainDashboard } from '../helpers/main';
import { interactAndWaitForResponses } from '../helpers/request';
import { getRegExpForTitleWithBranding } from '../helpers/title';
import { visit } from '../helpers/visit';

const apiReferencePath = '/main/apidocs';

const apiReferenceAlias = 'docs/swagger';

const requestConfig = {
    routeMatcherMap: {
        [apiReferenceAlias]: '/api/docs/swagger',
    },
};

const title = 'API Reference';

describe('API Reference', () => {
    withAuth();

    it('should visit via menu on top nav', () => {
        visitMainDashboard();

        interactAndWaitForResponses(() => {
            cy.get('button[aria-label="Help menu"').click();
            cy.get(`a:contains("${title}")`).click();
        }, requestConfig);

        cy.location('pathname').should('eq', apiReferencePath);
        cy.get(`h1:contains("${title}")`);
    });

    it('should visit via path', () => {
        visit(apiReferencePath, requestConfig);

        cy.get(`h1:contains("${title}")`);

        // Exception to pattern of separate test for title, because API Reference loads so slowly.
        cy.title().should('match', getRegExpForTitleWithBranding(title));
    });
});
