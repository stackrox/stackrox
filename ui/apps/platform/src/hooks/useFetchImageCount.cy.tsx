import {
    Label,
    LabelGroup,
    Masthead,
    MastheadBrand,
    MastheadLogo,
    MastheadMain,
    Page,
    PageSection,
    Title,
} from '@patternfly/react-core';

import BrandLogo from 'Components/PatternFly/BrandLogo';
import EntityTypeToggleGroup from 'Containers/Vulnerabilities/components/EntityTypeToggleGroup';
import DeploymentPageHeader from 'Containers/Vulnerabilities/WorkloadCves/Deployment/DeploymentPageHeader';
import type { DeploymentMetadata } from 'Containers/Vulnerabilities/WorkloadCves/Deployment/DeploymentPageHeader';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import useFetchDeploymentCountByQuery from './useFetchDeploymentCountByQuery';
import useFetchImageCount from './useFetchImageCount';

const scopedQuery = 'Severity:CRITICAL+Vulnerability State:OBSERVED';

const deploymentMetadata: DeploymentMetadata = {
    id: 'mock-deployment-001',
    name: 'cypress-test-deployment',
    namespace: 'stackrox',
    clusterName: 'remote',
    created: '2024-01-15T10:30:00Z',
    imageCount: 1,
};

function WorkloadCvesRestSurface() {
    const {
        data: imageCount,
        isLoading: imagesLoading,
        error: imagesError,
    } = useFetchImageCount(scopedQuery);
    const {
        data: deploymentCount,
        isLoading: deploymentsLoading,
        error: deploymentsError,
    } = useFetchDeploymentCountByQuery(scopedQuery);

    const isLoading = imagesLoading || deploymentsLoading;
    const error = imagesError ?? deploymentsError;

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
                <Title headingLevel="h1">Workload CVEs</Title>
                <p role="status">
                    {isLoading && 'Loading entity counts'}
                    {error && `Error: ${error.message}`}
                    {!isLoading &&
                        !error &&
                        `${imageCount ?? 'undefined'} images, ${deploymentCount ?? 'undefined'} deployments`}
                </p>
                <LabelGroup numLabels={1} className="pf-v6-u-my-md">
                    <Label>CVE severity: Critical</Label>
                </LabelGroup>
                {!isLoading &&
                    !error &&
                    imageCount !== undefined &&
                    deploymentCount !== undefined && (
                        <EntityTypeToggleGroup
                            entityTabs={['CVE', 'Image', 'Deployment']}
                            entityCounts={{
                                CVE: 12,
                                Image: imageCount,
                                Deployment: deploymentCount,
                            }}
                            onChange={() => {}}
                        />
                    )}
            </PageSection>
            <PageSection>
                <Title headingLevel="h2">Deployment</Title>
                <DeploymentPageHeader data={deploymentMetadata} />
            </PageSection>
        </Page>
    );
}

function mountSurface(
    reply:
        | { imageCount: number; deploymentCount: number; delay?: number }
        | { statusCode: number; body?: { message?: string }; delay?: number }
) {
    cy.intercept('GET', '/v1/imagescount*', (req) => {
        if ('statusCode' in reply) {
            req.reply({
                statusCode: reply.statusCode,
                delay: reply.delay,
                body: reply.body ?? { message: 'count failed' },
            });
            return;
        }
        req.reply({ delay: reply.delay, body: { count: reply.imageCount } });
    }).as('getImageCount');

    cy.intercept('GET', '/v1/deploymentscount*', (req) => {
        if ('statusCode' in reply) {
            req.reply({
                statusCode: reply.statusCode,
                delay: reply.delay,
                body: reply.body ?? { message: 'count failed' },
            });
            return;
        }
        req.reply({ delay: reply.delay, body: { count: reply.deploymentCount } });
    }).as('getDeploymentCount');

    cy.mount(
        <ComponentTestProvider>
            <WorkloadCvesRestSurface />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('loads scoped image and deployment counts from REST', () => {
        mountSurface({ imageCount: 7, deploymentCount: 4, delay: 100 });

        cy.findByRole('status').should('have.text', 'Loading entity counts');
        cy.screenshot('workload-cves-counts-loading');
        cy.wait('@getImageCount').then(({ request }) => {
            expect(decodeURIComponent(request.url)).to.include(`query=${scopedQuery}`);
        });
        cy.wait('@getDeploymentCount').then(({ request }) => {
            expect(decodeURIComponent(request.url)).to.include(`query=${scopedQuery}`);
        });
        cy.findByRole('status').should('have.text', '7 images, 4 deployments');
        cy.findByRole('button', { name: '12 CVEs' }).should('exist');
        cy.findByRole('button', { name: '7 Images' }).should('exist');
        cy.findByRole('button', { name: '4 Deployments' }).should('exist');
        cy.screenshot('workload-cves-list-counts');
    });

    it('shows deployment entity detail with a severity filter applied', () => {
        mountSurface({ imageCount: 7, deploymentCount: 4 });

        cy.wait(['@getImageCount', '@getDeploymentCount']);
        cy.findByRole('heading', { name: 'cypress-test-deployment' }).should('exist');
        cy.contains('In: remote/stackrox').should('exist');
        cy.contains('Images: 1').should('exist');
        cy.contains('CVE severity: Critical').should('exist');
        cy.screenshot('workload-cves-detail-and-filter');
    });

    it('surfaces a REST error', () => {
        mountSurface({ statusCode: 500, body: { message: 'count failed' } });

        cy.wait('@getImageCount');
        cy.findByRole('status').should('contain', 'Error');
        cy.screenshot('workload-cves-counts-error');
    });
});
