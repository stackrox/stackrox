import AiRiskSummaryCard from './AiRiskSummaryCard';

const summaryText = 'sync-worker scores 67/100 (Important).\nMain drivers: suspicious processes.';

describe(Cypress.spec.relative, () => {
    it('should show a spinner while the summary is loading', () => {
        cy.mount(
            <AiRiskSummaryCard summary={undefined} isLoading error={undefined} onClose={() => {}} />
        );

        cy.get('[aria-label="Generating AI risk summary"]').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('not.exist');
    });

    it('should render the summary with the review disclaimer on success', () => {
        cy.mount(
            <AiRiskSummaryCard
                summary={summaryText}
                isLoading={false}
                error={undefined}
                onClose={() => {}}
            />
        );

        cy.contains('AI risk briefing').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('exist');
        cy.contains('sync-worker scores 67/100 (Important).').should('exist');
        cy.get('[aria-label="Generating AI risk summary"]').should('not.exist');
    });

    it('should render an error alert when the request fails', () => {
        cy.mount(
            <AiRiskSummaryCard
                summary={undefined}
                isLoading={false}
                error={new Error('something went wrong')}
                onClose={() => {}}
            />
        );

        cy.contains('Unable to generate AI risk summary').should('exist');
        cy.contains('something went wrong').should('exist');
        cy.contains('Always review AI-generated content prior to use.').should('not.exist');
    });

    it('should copy the summary to the clipboard', () => {
        cy.mount(
            <AiRiskSummaryCard
                summary={summaryText}
                isLoading={false}
                error={undefined}
                onClose={() => {}}
            />
        );

        cy.get('button[aria-label="Copy AI summary to clipboard"]').click();

        cy.window().then((window) => {
            window.navigator.clipboard.readText().then((text) => {
                expect(text).to.eq(summaryText);
            });
        });
    });

    it('should invoke onClose when the close button is clicked', () => {
        const onClose = cy.stub().as('onClose');
        cy.mount(
            <AiRiskSummaryCard
                summary={summaryText}
                isLoading={false}
                error={undefined}
                onClose={onClose}
            />
        );

        cy.get('button[aria-label="Close AI investigation"]').click();
        cy.get('@onClose').should('have.been.calledOnce');
    });
});
