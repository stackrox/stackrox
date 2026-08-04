import ComponentTestProvider from 'test-utils/ComponentTestProvider';
import { riskWorkloadsBasePath } from 'routePaths';

import AdministrationEventDescription from './AdministrationEventDescription';

function makeEvent(resource, overrides) {
    return {
        id: 'test-event-id',
        type: 'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE',
        level: 'ADMINISTRATION_EVENT_LEVEL_ERROR',
        message: 'test error message',
        hint: 'test hint',
        domain: 'Image Scanning',
        resource,
        numOccurrences: '1',
        lastOccurredAt: '2024-01-01T00:00:00Z',
        createdAt: '2024-01-01T00:00:00Z',
        ...overrides,
    };
}

describe(Cypress.spec.relative, () => {
    it('should render image resource name as a link to the Risk page', () => {
        const event = makeEvent({
            type: 'Image',
            name: 'docker.io/library/nginx:latest',
            id: '',
        });

        cy.mount(
            <ComponentTestProvider>
                <AdministrationEventDescription event={event} />
            </ComponentTestProvider>
        );

        cy.findByText('docker.io/library/nginx:latest')
            .should('exist')
            .closest('a')
            .should('have.attr', 'href')
            .and('include', riskWorkloadsBasePath)
            .and('include', 'Image')
            .and('include', 'Full view');
    });

    it('should render image resource name as a link when both name and ID are present', () => {
        const event = makeEvent({
            type: 'Image',
            name: 'quay.io/rhacs-eng/qa:nginx-1.15',
            id: 'sha256:abc123',
        });

        cy.mount(
            <ComponentTestProvider>
                <AdministrationEventDescription event={event} />
            </ComponentTestProvider>
        );

        cy.findByText('quay.io/rhacs-eng/qa:nginx-1.15')
            .closest('a')
            .should('have.attr', 'href')
            .and('include', riskWorkloadsBasePath);

        cy.findByText('sha256:abc123').should('exist');
        cy.findByText('sha256:abc123').closest('a').should('not.exist');
    });

    it('should render non-Image resource name as plain text without a link', () => {
        const event = makeEvent({
            type: 'Cluster',
            name: 'production-cluster',
            id: 'cluster-123',
        });

        cy.mount(
            <ComponentTestProvider>
                <AdministrationEventDescription event={event} />
            </ComponentTestProvider>
        );

        cy.findByText('production-cluster').should('exist');
        cy.findByText('production-cluster').closest('a').should('not.exist');
    });

    it('should not render resource name row when name is empty', () => {
        const event = makeEvent({
            type: 'Image',
            name: '',
            id: 'sha256:abc123',
        });

        cy.mount(
            <ComponentTestProvider>
                <AdministrationEventDescription event={event} />
            </ComponentTestProvider>
        );

        cy.findByText('Resource name').should('not.exist');
        cy.findByText('sha256:abc123').should('exist');
    });
});
