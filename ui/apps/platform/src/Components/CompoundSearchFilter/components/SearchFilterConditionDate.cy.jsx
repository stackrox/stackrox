import { Toolbar, ToolbarContent } from '@patternfly/react-core';

import SearchFilterConditionDate from './SearchFilterConditionDate';

const attribute = {
    displayName: 'Discovered time',
    filterChipLabel: 'CVE discovered time',
    searchTerm: 'CVE Created Time',
    inputType: 'date-picker',
};

const selectors = {
    conditionSelectToggle: 'button[aria-label="Condition selector toggle"]',
    conditionSelectItems: '[aria-label="Condition selector menu"] li',
    singleDateInput: 'input[aria-label="Filter by date"]',
    startDateInput: 'input[aria-label="Filter by start date"]',
    endDateInput: 'input[aria-label="Filter by end date"]',
    applyButton: 'button[aria-label="Apply condition and date input to search"]',
};

const endBeforeStartErrorText = 'The end date must be on or after the start date';

function setup({ isBetweenEnabled = false } = {}) {
    const onSearch = cy.stub().as('onSearch');

    cy.mount(
        <Toolbar>
            <ToolbarContent>
                <SearchFilterConditionDate
                    attribute={attribute}
                    isBetweenEnabled={isBetweenEnabled}
                    onSearch={onSearch}
                />
            </ToolbarContent>
        </Toolbar>
    );
}

function selectCondition(condition) {
    cy.get(selectors.conditionSelectToggle).click();
    cy.get(`${selectors.conditionSelectItems} button:contains("${condition}")`)
        .filter((_, element) => {
            return Cypress.$(element).text().trim() === condition;
        })
        .click();
}

describe(Cypress.spec.relative, () => {
    describe('without isBetweenEnabled', () => {
        it('should not include Between in the condition selector', () => {
            setup({ isBetweenEnabled: false });

            cy.get(selectors.conditionSelectToggle).click();

            cy.get(selectors.conditionSelectItems).should('have.length', 3);
            cy.get(selectors.conditionSelectItems).eq(0).should('have.text', 'Before');
            cy.get(selectors.conditionSelectItems).eq(1).should('have.text', 'On');
            cy.get(selectors.conditionSelectItems).eq(2).should('have.text', 'After');
        });
    });

    describe('with isBetweenEnabled', () => {
        it('should include Between in the condition selector after Before/On/After', () => {
            setup({ isBetweenEnabled: true });

            cy.get(selectors.conditionSelectToggle).click();

            cy.get(selectors.conditionSelectItems).should('have.length', 4);
            cy.get(selectors.conditionSelectItems).eq(3).should('have.text', 'Between');
        });

        it('should keep single-date apply behavior for the After condition', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('After');

            cy.get(selectors.singleDateInput).type('01/15/2034');
            cy.get(selectors.applyButton).click();

            cy.get('@onSearch').should('have.been.calledWithExactly', [
                {
                    action: 'APPEND',
                    category: 'CVE Created Time',
                    value: '>01/15/2034',
                },
            ]);
            cy.get(selectors.singleDateInput).should('have.value', '');
        });

        it('should reveal start and end date inputs when Between is selected', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.singleDateInput).should('not.exist');
            cy.get(selectors.startDateInput).should('exist');
            cy.get(selectors.endDateInput).should('exist');
        });

        it('should disable the end date input until the start date is valid', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.endDateInput).should('be.disabled');

            cy.get(selectors.startDateInput).type('01/15/2034');

            cy.get(selectors.endDateInput).should('be.enabled');
            // End date defaults to the day after the start date.
            cy.get(selectors.endDateInput).should('have.value', '01/16/2034');
        });

        it('should apply a valid range as a tr/<startMs>-<endMs> value and clear the inputs', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.startDateInput).type('01/15/2034');
            cy.get(selectors.endDateInput).clear();
            cy.get(selectors.endDateInput).type('01/20/2034');

            cy.get(selectors.applyButton).click();

            const startMs = new Date(2034, 0, 15, 0, 0, 0, 0).getTime();
            const endMs = new Date(2034, 0, 20, 23, 59, 59, 999).getTime();
            cy.get('@onSearch').should('have.been.calledWithExactly', [
                {
                    action: 'APPEND',
                    category: 'CVE Created Time',
                    value: `tr/${startMs}-${endMs}`,
                },
            ]);
            cy.get(selectors.startDateInput).should('have.value', '');
            cy.get(selectors.endDateInput).should('have.value', '');
        });

        it('should apply a same-day range', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.startDateInput).type('01/15/2034');
            cy.get(selectors.endDateInput).clear();
            cy.get(selectors.endDateInput).type('01/15/2034');

            cy.get(selectors.applyButton).click();

            const startMs = new Date(2034, 0, 15, 0, 0, 0, 0).getTime();
            const endMs = new Date(2034, 0, 15, 23, 59, 59, 999).getTime();
            cy.get('@onSearch').should('have.been.calledWithExactly', [
                {
                    action: 'APPEND',
                    category: 'CVE Created Time',
                    value: `tr/${startMs}-${endMs}`,
                },
            ]);
        });

        it('should show an inline error and not emit when the end date is before the start date', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.startDateInput).type('01/15/2034');
            cy.get(selectors.endDateInput).clear();
            cy.get(selectors.endDateInput).type('01/10/2034');

            cy.contains(endBeforeStartErrorText).should('exist');

            cy.get(selectors.applyButton).click();

            cy.get('@onSearch').should('not.have.been.called');
        });

        it('should not emit when the end date is empty', () => {
            setup({ isBetweenEnabled: true });

            selectCondition('Between');

            cy.get(selectors.startDateInput).type('01/15/2034');
            cy.get(selectors.endDateInput).clear();

            cy.get(selectors.applyButton).click();

            cy.get('@onSearch').should('not.have.been.called');
        });
    });
});
