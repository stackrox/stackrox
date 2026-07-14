import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { assertCannotFindThePage, visit } from '../../helpers/visit';
import navSelectors from '../../selectors/navigation';

describe('Risk - Secrets page', () => {
    withAuth();

    describe('with ROX_UI_SECRETS_PAGE_MIGRATION feature flag enabled', () => {
        before(function () {
            if (!hasFeatureFlag('ROX_UI_SECRETS_PAGE_MIGRATION')) {
                this.skip();
            }
        });

        it('should render the Secrets page with the correct heading', () => {
            visit('/main/risk/secrets');

            cy.get('h1:contains("Secrets")');
        });

        it('should show Risk as an expandable nav section with Workloads and Secrets children', () => {
            visit('/main/risk/secrets');

            cy.get(`${navSelectors.navExpandable}:contains("Risk")`);
            cy.get(`${navSelectors.nestedNavLinks}:contains("Workloads")`);
            cy.get(`${navSelectors.nestedNavLinks}:contains("Secrets")`).should(
                'have.class',
                'pf-m-current'
            );
        });
    });

    describe('without ROX_UI_SECRETS_PAGE_MIGRATION feature flag', () => {
        before(function () {
            if (hasFeatureFlag('ROX_UI_SECRETS_PAGE_MIGRATION')) {
                this.skip();
            }
        });

        it('should not render the Secrets page', () => {
            visit('/main/risk/secrets');

            assertCannotFindThePage();
        });

        it('should show Risk as a plain nav link, not an expandable section', () => {
            visit('/main/risk/workloads');

            cy.get(`${navSelectors.navLinks}:contains("Risk")`);
            cy.get(`${navSelectors.navExpandable}:contains("Risk")`).should('not.exist');
        });
    });
});
