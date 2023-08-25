import withAuth from '../../helpers/basicAuth';
import { visitSystemHealth, visitSystemHealthFromLeftNav } from '../../helpers/systemHealth';
import { getRegExpForTitleWithBranding } from '../../helpers/title';
import navSelectors from '../../selectors/navigation';

describe('System Health', () => {
    withAuth();

    it('should visit from left nav', () => {
        visitSystemHealthFromLeftNav();
    });

    it('should have selected item in left nav', () => {
        visitSystemHealth();

        cy.get(`${navSelectors.navExpandable}:contains("Platform Configuration")`);
        cy.get(`${navSelectors.nestedNavLinks}:contains("System Health")`).should(
            'have.class',
            'pf-m-current'
        );
    });

    it('should have title', () => {
        visitSystemHealth();

        cy.title().should('match', getRegExpForTitleWithBranding('System Health'));
    });
});
