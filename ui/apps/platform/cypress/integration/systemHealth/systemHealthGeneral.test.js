import withAuth from '../../helpers/basicAuth';
import { visitSystemHealth, visitSystemHealthFromLeftNav } from '../../helpers/systemHealth';
import navSelectors from '../../selectors/navigation';

describe('System Health general', () => {
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
});
