import {
    Content,
    Flex,
    FlexItem,
    Masthead,
    MastheadBrand,
    MastheadLogo,
    MastheadMain,
    Page,
    PageSection,
    Title,
} from '@patternfly/react-core';

import BrandLogo from 'Components/PatternFly/BrandLogo';
import { NetworkGraphURLStateProvider } from 'Containers/NetworkGraph/NetworkGraphURLStateContext';
import NetworkPoliciesGenerationScope from 'Containers/NetworkGraph/simulation/NetworkPoliciesGenerationScope';
import type { NetworkScopeHierarchy } from 'Containers/NetworkGraph/types/networkScopeHierarchy';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import type { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import useFetchDeploymentCount from './useFetchDeploymentCount';

const searchFilter: SearchFilter = {
    Cluster: 'production',
    Namespace: 'stackrox',
};

const scopeHierarchy: NetworkScopeHierarchy = {
    cluster: { id: 'cluster-1', name: 'production' },
    namespaces: ['stackrox'],
    deployments: [],
    remainingQuery: {},
};

function DeploymentCountSurface({ filter }: { filter: SearchFilter }) {
    const { data, isLoading, error } = useFetchDeploymentCount(filter);

    return (
        <Page
            masthead={
                <Masthead>
                    <MastheadMain>
                        <MastheadBrand>
                            <MastheadLogo>
                                <BrandLogo />
                            </MastheadLogo>
                        </MastheadBrand>
                    </MastheadMain>
                </Masthead>
            }
        >
            <PageSection>
                <Title headingLevel="h1">Network Graph</Title>
                <p role="status">
                    {isLoading && 'Loading deployment count'}
                    {error && `Error: ${error.message}`}
                    {!isLoading && !error && `${data ?? 'undefined'} deployments`}
                </p>
                {!isLoading && !error && data !== undefined && (
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                        className="pf-v6-u-p-md"
                    >
                        <FlexItem>
                            <Title headingLevel="h2">Generated network policies</Title>
                            <Content component="p" className="pf-v6-u-font-weight-bold">
                                Scope of baseline:
                            </Content>
                        </FlexItem>
                        <NetworkGraphURLStateProvider>
                            <NetworkPoliciesGenerationScope
                                scopeHierarchy={scopeHierarchy}
                                scopeDeploymentCount={data}
                            />
                        </NetworkGraphURLStateProvider>
                    </Flex>
                )}
            </PageSection>
        </Page>
    );
}

function mountSurface(
    reply:
        | { count: number; delay?: number }
        | { statusCode: number; body?: { message?: string }; delay?: number }
) {
    cy.intercept('GET', '/v1/deploymentscount*', (req) => {
        if ('statusCode' in reply) {
            req.reply({
                statusCode: reply.statusCode,
                delay: reply.delay,
                body: reply.body ?? { message: 'count failed' },
            });
            return;
        }
        req.reply({ delay: reply.delay, body: { count: reply.count } });
    }).as('getDeploymentCount');

    cy.mount(
        <ComponentTestProvider>
            <DeploymentCountSurface filter={searchFilter} />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('loads a scoped deployment count from REST', () => {
        mountSurface({ count: 12, delay: 100 });

        cy.findByRole('status').should('have.text', 'Loading deployment count');
        cy.screenshot('deployment-count-loading');
        cy.wait('@getDeploymentCount').then(({ request }) => {
            const expectedQuery = getRequestQueryStringForSearchFilter(searchFilter);
            expect(decodeURIComponent(request.url)).to.include(`query=${expectedQuery}`);
        });
        cy.findByRole('status').should('have.text', '12 deployments');
        cy.findByRole('button', { name: '12 deployments' }).should('exist');
        cy.screenshot('deployment-count-happy');
    });

    it('shows an empty zero count', () => {
        mountSurface({ count: 0 });

        cy.wait('@getDeploymentCount');
        cy.findByRole('status').should('have.text', '0 deployments');
        cy.findByRole('button', { name: '0 deployments' }).should('exist');
        cy.screenshot('deployment-count-zero');
    });

    it('surfaces a REST error', () => {
        mountSurface({ statusCode: 500, body: { message: 'count failed' } });

        cy.wait('@getDeploymentCount');
        cy.findByRole('status').should('contain', 'Error');
        cy.screenshot('deployment-count-error');
    });
});
