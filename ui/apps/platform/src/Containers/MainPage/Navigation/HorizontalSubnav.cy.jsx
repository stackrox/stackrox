import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import HorizontalSubnav from './HorizontalSubnav';

// Mock the permission and feature flag hooks
const mockRoutePredicates = {
    hasReadAccess: () => true,
    isFeatureFlagEnabled: () => true,
};

function setup(pathname) {
    window.history.pushState({}, document.title, pathname);

    cy.mount(
        <ComponentTestProvider>
            <HorizontalSubnav
                hasReadAccess={mockRoutePredicates.hasReadAccess}
                isFeatureFlagEnabled={mockRoutePredicates.isFeatureFlagEnabled}
            />
        </ComponentTestProvider>
    );
}

describe(Cypress.spec.relative, () => {
    it('should render the violations subnav when on violations page', () => {
        setup('/main/violations');

        // Verify that the violations subnav items are rendered
        cy.contains('User Workloads').should('exist');
        cy.contains('Platform').should('exist');
        cy.contains('All Violations').should('exist');
    });

    it('should not render subnav when not on a violations or vulnerabilities page', () => {
        setup('/main/dashboard');

        // The component should not render anything when not on a relevant page
        cy.get('nav').should('not.exist');
    });

    describe('Violations active state behavior', () => {
        it('should default User Workloads to active when no filter is present', () => {
            setup('/main/violations');

            // Verify User Workloads is active by default when no filter is present
            cy.findByRole('link', { name: 'User Workloads' }).should('have.class', 'pf-m-current');

            // Verify other links are not active
            cy.findByRole('link', { name: 'Platform' }).should('not.have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'All Violations' }).should(
                'not.have.class',
                'pf-m-current'
            );
        });

        it('should work with additional URL parameters', () => {
            setup(
                '/main/violations?someParam=value&filteredWorkflowView=Applications view&anotherParam=test'
            );

            // Check that User Workloads link is still active with additional params
            cy.findByRole('link', { name: 'User Workloads' }).should('have.class', 'pf-m-current');
        });

        it('should update active state when clicking through navigation items', () => {
            setup('/main/violations');

            // Initially User Workloads should be active by default
            cy.findByRole('link', { name: 'User Workloads' }).should('have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'Platform' }).should('not.have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'All Violations' }).should(
                'not.have.class',
                'pf-m-current'
            );

            // Click on Platform and verify it becomes active
            cy.findByRole('link', { name: 'Platform' }).click();
            cy.findByRole('link', { name: 'Platform' }).should('have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'User Workloads' }).should(
                'not.have.class',
                'pf-m-current'
            );
            cy.findByRole('link', { name: 'All Violations' }).should(
                'not.have.class',
                'pf-m-current'
            );

            // Click on All Violations and verify it becomes active
            cy.findByRole('link', { name: 'All Violations' }).click();
            cy.findByRole('link', { name: 'All Violations' }).should('have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'User Workloads' }).should(
                'not.have.class',
                'pf-m-current'
            );
            cy.findByRole('link', { name: 'Platform' }).should('not.have.class', 'pf-m-current');

            // Click back on User Workloads (first item) and verify it becomes active again
            cy.findByRole('link', { name: 'User Workloads' }).click();
            cy.findByRole('link', { name: 'User Workloads' }).should('have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'Platform' }).should('not.have.class', 'pf-m-current');
            cy.findByRole('link', { name: 'All Violations' }).should(
                'not.have.class',
                'pf-m-current'
            );
        });
    });
});
