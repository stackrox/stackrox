import withAuth from '../../helpers/basicAuth';
import { hasFeatureFlag } from '../../helpers/features';
import { interceptAndWatchRequests } from '../../helpers/request';
import { sortByTableHeader } from '../../helpers/tableHelpers';
import { visit } from '../../helpers/visit';
import navSelectors from '../../selectors/navigation';

const listSecretsAlias = 'listSecrets';
const secretsCountAlias = 'secretsCount';

const routeMatcherMapForSecrets = {
    [listSecretsAlias]: {
        method: 'GET' as const,
        url: '/v1/secretsextended?*',
    },
    [secretsCountAlias]: {
        method: 'GET' as const,
        url: '/v1/secretscount?*',
    },
};

function visitSecretsPage(
    routeMatcherMap?: Record<string, { method: string; url: string }>,
    staticResponseMap?: Record<string, { body?: unknown; fixture?: string }>
) {
    return visit('/main/risk/secrets', routeMatcherMap, staticResponseMap);
}

describe('Risk - Secrets page', () => {
    withAuth();

    describe('with ROX_UI_SECRETS_PAGE_MIGRATION feature flag enabled', () => {
        before(function () {
            if (!hasFeatureFlag('ROX_UI_SECRETS_PAGE_MIGRATION')) {
                this.skip();
            }
        });

        it('should render the heading and nav structure', () => {
            visitSecretsPage(routeMatcherMapForSecrets);

            cy.get('h1').contains('Secrets Risk');

            cy.get(`${navSelectors.navExpandable}:contains("Risk")`);
            cy.get(`${navSelectors.nestedNavLinks}:contains("Workloads")`);
            cy.get(`${navSelectors.nestedNavLinks}:contains("Secrets")`).should(
                'have.class',
                'pf-m-current'
            );
        });

        it('should sort by table columns', () => {
            interceptAndWatchRequests(routeMatcherMapForSecrets).then(({ waitForRequests }) => {
                visitSecretsPage();
                waitForRequests();

                sortByTableHeader('Secret');
                cy.wait(`@${listSecretsAlias}`).then((interception) => {
                    expect(interception.request.url).to.include('Secret');
                });

                sortByTableHeader('Cluster');
                cy.wait(`@${listSecretsAlias}`).then((interception) => {
                    expect(interception.request.url).to.include('Cluster');
                });

                sortByTableHeader('Namespace');
                cy.wait(`@${listSecretsAlias}`).then((interception) => {
                    expect(interception.request.url).to.include('Namespace');
                });

                sortByTableHeader('Created');
                cy.wait(`@${listSecretsAlias}`).then((interception) => {
                    expect(interception.request.url).to.include('Created%20Time');
                });
            });
        });

        it('should display an empty state when no secrets are returned', () => {
            visitSecretsPage(routeMatcherMapForSecrets, {
                [listSecretsAlias]: { body: { secrets: [] } },
                [secretsCountAlias]: { body: { count: 0 } },
            });

            cy.get('tbody').contains('No results found');
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

            // Without the feature flag, 'secrets' is interpreted as a deployment ID and the page will show an error
            cy.get('h2').contains(`deployment with id 'secrets' does not exist`);
        });

        it('should show Risk as a plain nav link, not an expandable section', () => {
            visit('/main/risk/workloads');

            cy.get(`${navSelectors.navLinks}:contains("Risk")`);
            cy.get(`${navSelectors.navExpandable}:contains("Risk")`).should('not.exist');
        });
    });
});
