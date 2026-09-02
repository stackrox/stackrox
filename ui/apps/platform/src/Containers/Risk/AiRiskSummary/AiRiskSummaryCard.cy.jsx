import AiRiskSummaryCard from './AiRiskSummaryCard';

const summaryText = 'sync-worker scores 67/100 (Important).\nMain drivers: suspicious processes.';

function setup(props = {}) {
    cy.mount(
        <AiRiskSummaryCard
            summary={undefined}
            isLoading={false}
            error={undefined}
            isExpanded
            onExpand={() => {}}
            onRetry={() => {}}
            {...props}
        />
    );
}

describe(Cypress.spec.relative, () => {
    it('should show a spinner while the summary is loading', () => {
        setup({ isLoading: true });

        cy.get('[aria-label="Generating AI risk summary"]').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('not.exist');
    });

    it('should render an error alert with a retry action when the request fails', () => {
        const onRetry = cy.stub().as('onRetry');
        setup({ error: new Error('something went wrong'), onRetry });

        cy.contains('Unable to generate AI risk summary').should('exist');
        cy.contains('something went wrong').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('not.exist');

        cy.contains('button', 'Try again').click();
        cy.get('@onRetry').should('have.been.calledOnce');
    });

    it('should render the summary with the review disclaimer on success', () => {
        setup({ summary: summaryText });

        cy.contains('AI risk briefing').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('exist');
        cy.contains('sync-worker scores 67/100 (Important).').should('exist');
        cy.get('[aria-label="Generating AI risk summary"]').should('not.exist');
    });

    it('should copy the summary to the clipboard', () => {
        setup({ summary: summaryText });

        cy.get('button[aria-label="Copy AI summary to clipboard"]').click();

        cy.window().then((window) => {
            window.navigator.clipboard.readText().then((text) => {
                expect(text).to.eq(summaryText);
            });
        });
    });

    it('should invoke onExpand when the collapse toggle is clicked', () => {
        const onExpand = cy.stub().as('onExpand');
        setup({ summary: summaryText, onExpand });

        cy.get('button[aria-label="Collapse AI risk briefing"]').click();
        cy.get('@onExpand').should('have.been.calledOnce');
    });

    it('should hide the summary body when collapsed', () => {
        setup({ summary: summaryText, isExpanded: false });

        // The title remains, but the collapsed card removes its body from the DOM.
        cy.contains('AI risk briefing').should('exist');
        cy.contains('sync-worker scores 67/100 (Important).').should('not.exist');
        cy.get('button[aria-label="Expand AI risk briefing"]').should('exist');
    });
});
