import { FeatureFlagsContext } from 'hooks/useFeatureFlags';
import ComponentTestProvider from 'test-utils/ComponentTestProvider';

import SearchTable from './SearchTable';

// The global search API has no server-side pagination, so SearchTable paginates on the client.
// These tests verify that only one page of rows renders at a time and that paging/per-page work.

const MockFeatureFlagsProvider = ({ children }) => (
    <FeatureFlagsContext.Provider
        value={{
            isFeatureFlagEnabled: () => false,
            isLoadingFeatureFlags: false,
            error: undefined,
        }}
    >
        {children}
    </FeatureFlagsContext.Provider>
);

function makeResults(count, { category = 'DEPLOYMENTS', prefix = 'dep' } = {}) {
    return Array.from({ length: count }, (_, index) => {
        const name = `${prefix}-${String(index).padStart(3, '0')}`;
        return { id: name, name, category, fieldToMatches: {}, score: 0, location: '' };
    });
}

function setup(props, url = '/main/search') {
    // Reset the URL so pagination parameters start unset for each test.
    window.history.pushState({}, '', url);
    cy.mount(
        <ComponentTestProvider>
            <MockFeatureFlagsProvider>
                <SearchTable
                    navCategory="SEARCH_UNSET"
                    searchFilter={{}}
                    searchResults={[]}
                    {...props}
                />
            </MockFeatureFlagsProvider>
        </ComponentTestProvider>
    );
}

const nextPageButton = 'button[aria-label="Go to next page"]';

describe(Cypress.spec.relative, () => {
    it('renders only the first page of results by default', () => {
        setup({ searchResults: makeResults(45) });

        cy.get('tbody tr').should('have.length', 20);
        cy.get('tbody tr').first().should('contain.text', 'dep-000');
        cy.contains('button', '1 - 20 of 45').should('exist');
    });

    it('shows the next page of results when paginating', () => {
        setup({ searchResults: makeResults(45) });

        cy.get(nextPageButton).first().click();

        cy.get('tbody tr').should('have.length', 20);
        cy.get('tbody tr').first().should('contain.text', 'dep-020');
        cy.contains('button', '21 - 40 of 45').should('exist');
    });

    it('shows the remaining results on the last page', () => {
        setup({ searchResults: makeResults(45) });

        cy.get(nextPageButton).first().click();
        cy.get(nextPageButton).first().click();

        cy.get('tbody tr').should('have.length', 5);
        cy.get('tbody tr').first().should('contain.text', 'dep-040');
    });

    it('renders more results after increasing items per page', () => {
        setup({ searchResults: makeResults(45) });

        // The per-page toggle displays the range summary; open it and choose 50 per page.
        cy.contains('button', '1 - 20 of 45').click();
        cy.get('button[role="menuitem"]').contains('50 per page').click();

        cy.get('tbody tr').should('have.length', 45);
    });

    it('clamps an out-of-range page from the URL and corrects the URL', () => {
        // 45 results at 20 per page => 3 pages; a bookmarked page=5 is out of range.
        setup({ searchResults: makeResults(45) }, '/main/search?page=5');

        // The table shows the last page (rows 41 - 45) instead of an empty page.
        cy.get('tbody tr').should('have.length', 5);
        cy.contains('button', '41 - 45 of 45').should('exist');

        // The URL is corrected to the last page rather than left pointing beyond the range.
        cy.location('search').should('contain', 'page=3');
        cy.location('search').should('not.contain', 'page=5');
    });

    it('limits results to the active category before paginating', () => {
        setup({
            navCategory: 'DEPLOYMENTS',
            searchResults: [
                ...makeResults(25, { category: 'DEPLOYMENTS', prefix: 'dep' }),
                ...makeResults(10, { category: 'IMAGES', prefix: 'img' }),
            ],
        });

        cy.get('tbody tr').should('have.length', 20);
        cy.contains('button', '1 - 20 of 25').should('exist');
        cy.get('tbody tr').each(($tr) => {
            cy.wrap($tr).should('contain.text', 'dep-');
        });
    });
});
