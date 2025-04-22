import React from 'react';
import { SummaryCard, SummaryCardLayout } from './SummaryCardLayout';

describe(Cypress.spec.relative, () => {
    it('should render an error alert if an error is provided', () => {
        cy.mount(
            <SummaryCardLayout
                errorAlertTitle="This is an error in a test"
                error={new Error('An error occurred')}
                isLoading
            >
                <SummaryCard
                    loadingText="Loading..."
                    data={{ key: 'does not render' }}
                    renderer={({ data }) => <div>{data.key}</div>}
                />
            </SummaryCardLayout>
        );

        cy.findByText('Danger alert:').should('exist');
        cy.findByText('This is an error in a test').should('exist');
        cy.findByText('An error occurred').should('exist');
        cy.findByText('Loading...').should('not.exist');
        cy.findByText('does not render').should('not.exist');
    });

    it('should render a loading skeleton instead of content when in a loading state', () => {
        cy.mount(
            <SummaryCardLayout isLoading>
                <SummaryCard
                    loadingText="Loading..."
                    data={{ key: 'does not render' }}
                    renderer={({ data }) => <div>{data.key}</div>}
                />
            </SummaryCardLayout>
        );

        cy.findByText('Danger alert:').should('not.exist');
        cy.findByText('Loading...').should('exist');
        cy.findByText('does not render').should('not.exist');
    });

    it('should render a loading skeleton if data is not provided', () => {
        cy.mount(
            <SummaryCardLayout isLoading={false}>
                <SummaryCard
                    loadingText="Loading..."
                    data={null}
                    renderer={({ data }) => <div>{data.key}</div>}
                />
            </SummaryCardLayout>
        );

        cy.findByText('Danger alert:').should('not.exist');
        cy.findByText('Loading...').should('exist');
        cy.findByText('does not render').should('not.exist');
    });

    it('should render the provided data', () => {
        cy.mount(
            <SummaryCardLayout isLoading={false}>
                <SummaryCard
                    loadingText="Loading..."
                    data={{ key: 'does render' }}
                    renderer={({ data }) => <div>{data.key}</div>}
                />
            </SummaryCardLayout>
        );

        cy.findByText('Danger alert:').should('not.exist');
        cy.findByText('Loading...').should('not.exist');
        cy.findByText('does render').should('exist');
    });
});
