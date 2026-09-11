import withAuth from '../../helpers/basicAuth';
import { visit } from '../../helpers/visit';
import selectors from './ListeningEndpoints.selectors';

const listeningEndpointsPath = '/main/listening-endpoints';

const deploymentsRouteMatcherMap = {
    deployments: { method: 'GET', url: '/v1/deployments?*' },
    deploymentsCount: { method: 'GET', url: '/v1/deploymentscount?*' },
};

function makeEndpoint(deploymentId, port) {
    return {
        endpoint: { port, protocol: 'L4_PROTOCOL_TCP' },
        clusterId: 'test-cluster-id',
        namespace: 'default',
        deploymentId,
        imageId: 'test-image-id',
        containerName: 'listeners',
        podId: `pod-${deploymentId}`,
        podUid: `uid-${deploymentId}`,
        signal: {
            id: `signal-${deploymentId}-${port}`,
            containerId: `container-${deploymentId}`,
            time: '2026-08-06T00:00:00Z',
            pid: 1000 + port,
            uid: 0,
            gid: 0,
            lineage: [],
            scraped: false,
            lineageInfo: [],
            execFilePath: '/usr/bin/python3',
        },
        containerStartTime: '2026-08-06T00:00:00Z',
    };
}

const endpointsByDeployment = {
    'deployment-a-id': Array.from({ length: 50 }, (_, i) =>
        makeEndpoint('deployment-a-id', 8000 + i)
    ),
    'deployment-b-id': Array.from({ length: 50 }, (_, i) =>
        makeEndpoint('deployment-b-id', 9000 + i)
    ),
};

const deploymentResponse = {
    deployments: [
        {
            id: 'deployment-a-id',
            name: 'listeners-alpha',
            namespace: 'default',
            cluster: 'test-cluster',
            clusterId: 'test-cluster-id',
        },
        {
            id: 'deployment-b-id',
            name: 'listeners-beta',
            namespace: 'default',
            cluster: 'test-cluster',
            clusterId: 'test-cluster-id',
        },
    ],
};

function interceptListeningEndpointsAPI() {
    cy.intercept('GET', '/v1/listening_endpoints/deployment/*', (req) => {
        const url = new URL(req.url, window.location.origin);
        const deploymentId = url.pathname.split('/').pop();
        const offset = parseInt(url.searchParams.get('pagination.offset') || '0', 10);
        const limit = parseInt(url.searchParams.get('pagination.limit') || '0', 10);

        const allEndpoints = endpointsByDeployment[deploymentId] || [];
        const page = limit > 0 ? allEndpoints.slice(offset, offset + limit) : allEndpoints;
        req.reply({
            body: {
                listeningEndpoints: page,
                totalListeningEndpoints: allEndpoints.length,
            },
        });
    }).as('listeningEndpoints');
}

describe('Listening endpoints pagination within a deployment', () => {
    withAuth();

    beforeEach(() => {
        interceptListeningEndpointsAPI();
    });

    it('should paginate listening endpoints independently for each deployment', () => {
        visit(listeningEndpointsPath, deploymentsRouteMatcherMap, {
            deployments: { body: deploymentResponse },
            deploymentsCount: { body: { count: 2 } },
        });

        cy.get('h1:contains("Listening endpoints")');

        // Assert that two deployment rows are displayed
        cy.get(`${selectors.deploymentTable} > tbody`).should('have.length', 2);

        // Both deployments should show a count of 50
        const rowSelectorA = `${selectors.deploymentTable} > tbody:has(${selectors.tableRowWithValueForColumn('Deployment', 'listeners-alpha')})`;
        const rowSelectorB = `${selectors.deploymentTable} > tbody:has(${selectors.tableRowWithValueForColumn('Deployment', 'listeners-beta')})`;

        cy.get(`${rowSelectorA} ${selectors.tableRowWithValueForColumn('Count', '50')}`);
        cy.get(`${rowSelectorB} ${selectors.tableRowWithValueForColumn('Count', '50')}`);

        // Expand deployment A
        cy.get(`${rowSelectorA} ${selectors.expandableRowToggle}`).click();

        const processTableA = `${rowSelectorA} ${selectors.processTable}`;
        const expandedContentA = `${rowSelectorA} .pf-v6-c-card`;

        // Assert that the first page shows 20 rows
        cy.get(`${processTableA} > tbody > tr`).should('have.length', 20);

        // Verify deployment A page 1 starts with port 8000
        cy.get(`${processTableA} ${selectors.tableRowWithValueForColumn('Port', '8000')}`).should(
            'exist'
        );

        // Navigate deployment A to page 2
        cy.get(expandedContentA).find('[data-action="next"]').click();
        cy.wait('@listeningEndpoints');
        cy.get(`${processTableA} > tbody > tr`).should('have.length', 20);

        // Deployment A page 2 should start with port 8020, not 8000
        cy.get(`${processTableA} ${selectors.tableRowWithValueForColumn('Port', '8020')}`).should(
            'exist'
        );
        cy.get(`${processTableA} ${selectors.tableRowWithValueForColumn('Port', '8000')}`).should(
            'not.exist'
        );

        // Now expand deployment B — it should start on page 1 independently
        cy.get(`${rowSelectorB} ${selectors.expandableRowToggle}`).click();

        const processTableB = `${rowSelectorB} ${selectors.processTable}`;

        // Deployment B should show page 1 starting with port 9000
        cy.get(`${processTableB} > tbody > tr`).should('have.length', 20);
        cy.get(`${processTableB} ${selectors.tableRowWithValueForColumn('Port', '9000')}`).should(
            'exist'
        );

        // Deployment A should still be on page 2 (port 8020 present, port 8000 absent)
        cy.get(`${processTableA} ${selectors.tableRowWithValueForColumn('Port', '8020')}`).should(
            'exist'
        );
        cy.get(`${processTableA} ${selectors.tableRowWithValueForColumn('Port', '8000')}`).should(
            'not.exist'
        );
    });

    it('should navigate to the last page and disable the next button', () => {
        visit(listeningEndpointsPath, deploymentsRouteMatcherMap, {
            deployments: { body: deploymentResponse },
            deploymentsCount: { body: { count: 2 } },
        });

        cy.get('h1:contains("Listening endpoints")');

        const rowSelector = `${selectors.deploymentTable} > tbody:has(${selectors.tableRowWithValueForColumn('Deployment', 'listeners-alpha')})`;

        // Expand deployment A
        cy.get(`${rowSelector} ${selectors.expandableRowToggle}`).click();

        const processTable = `${rowSelector} ${selectors.processTable}`;
        const expandedContent = `${rowSelector} .pf-v6-c-card`;

        // Page 1: 20 rows
        cy.get(`${processTable} > tbody > tr`).should('have.length', 20);

        // Page 2: 20 rows
        cy.get(expandedContent).find('[data-action="next"]').click();
        cy.wait('@listeningEndpoints');
        cy.get(`${processTable} > tbody > tr`).should('have.length', 20);

        // Page 3 (last): 10 rows
        cy.get(expandedContent).find('[data-action="next"]').click();
        cy.wait('@listeningEndpoints');
        cy.get(`${processTable} > tbody > tr`).should('have.length', 10);

        // Next button should be disabled
        cy.get(expandedContent).find('[data-action="next"]').should('be.disabled');

        // Navigate back to page 2
        cy.get(expandedContent).find('[data-action="previous"]').click();
        cy.wait('@listeningEndpoints');
        cy.get(`${processTable} > tbody > tr`).should('have.length', 20);
    });
});
