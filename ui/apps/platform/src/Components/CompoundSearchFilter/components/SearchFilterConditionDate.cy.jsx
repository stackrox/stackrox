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
    relativeDaysInput: 'input[aria-label="Number of days"]',
    relativeApplyButton: 'button[aria-label="Apply relative date filter"]',
    relativeMinDaysInput: 'input[aria-label="Minimum days ago"]',
    relativeMaxDaysInput: 'input[aria-label="Maximum days ago"]',
    relativeRangeApplyButton: 'button[aria-label="Apply relative date range filter"]',
};

const endBeforeStartErrorText = 'The end date must be on or after the start date';

function setup() {
    const onSearch = cy.stub().as('onSearch');

    cy.mount(
        <Toolbar>
            <ToolbarContent>
                <SearchFilterConditionDate attribute={attribute} onSearch={onSearch} />
            </ToolbarContent>
        </Toolbar>
    );
}

function selectCondition(condition) {
    cy.get(selectors.conditionSelectToggle).click();
    cy.get(`${selectors.conditionSelectItems} button:contains("${condition}")`).click();
}

describe(Cypress.spec.relative, () => {
    it('should list all conditions in the expected order', () => {
        setup();

        cy.get(selectors.conditionSelectToggle).click();

        cy.get(selectors.conditionSelectItems).should('have.length', 6);
        cy.get(selectors.conditionSelectItems).eq(0).should('have.text', 'Before');
        cy.get(selectors.conditionSelectItems).eq(1).should('have.text', 'On');
        cy.get(selectors.conditionSelectItems).eq(2).should('have.text', 'After');
        cy.get(selectors.conditionSelectItems).eq(3).should('have.text', 'More than (days ago)');
        cy.get(selectors.conditionSelectItems).eq(4).should('have.text', 'Between');
        cy.get(selectors.conditionSelectItems).eq(5).should('have.text', 'Between (days ago)');
    });

    it('should keep single-date apply behavior for the After condition', () => {
        setup();

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

    it('should apply the On condition as a bare date with no prefix', () => {
        setup();

        selectCondition('On');

        cy.get(selectors.singleDateInput).type('01/15/2034');
        cy.get(selectors.applyButton).click();

        cy.get('@onSearch').should('have.been.calledWithExactly', [
            {
                action: 'APPEND',
                category: 'CVE Created Time',
                value: '01/15/2034',
            },
        ]);
        cy.get(selectors.singleDateInput).should('have.value', '');
    });

    it('should reveal start and end date inputs when Between is selected', () => {
        setup();

        selectCondition('Between');

        cy.get(selectors.singleDateInput).should('not.exist');
        cy.get(selectors.startDateInput).should('exist');
        cy.get(selectors.endDateInput).should('exist');
    });

    it('should disable the end date input until the start date is valid', () => {
        setup();

        selectCondition('Between');

        cy.get(selectors.endDateInput).should('be.disabled');

        cy.get(selectors.startDateInput).type('01/15/2034');

        cy.get(selectors.endDateInput).should('be.enabled');
        // End date defaults to the day after the start date.
        cy.get(selectors.endDateInput).should('have.value', '01/16/2034');
    });

    it('should keep a chosen end date when the start date changes to an earlier date', () => {
        setup();

        selectCondition('Between');

        cy.get(selectors.startDateInput).type('01/15/2034');
        cy.get(selectors.endDateInput).clear();
        cy.get(selectors.endDateInput).type('03/20/2034');

        cy.get(selectors.startDateInput).clear();
        cy.get(selectors.startDateInput).type('01/10/2034');

        cy.get(selectors.endDateInput).should('have.value', '03/20/2034');
    });

    it('should re-default the end date when the start date moves past it', () => {
        setup();

        selectCondition('Between');

        cy.get(selectors.startDateInput).type('01/15/2034');
        // The defaulted end date is before the new start date below.
        cy.get(selectors.endDateInput).should('have.value', '01/16/2034');
        cy.get(selectors.startDateInput).clear();
        cy.get(selectors.startDateInput).type('02/01/2034');

        cy.get(selectors.endDateInput).should('have.value', '02/02/2034');
    });

    it('should apply a valid range as a tr/<startMs>-<endMs> value and clear the inputs', () => {
        setup();

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
        setup();

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
        setup();

        selectCondition('Between');

        cy.get(selectors.startDateInput).type('01/15/2034');
        cy.get(selectors.endDateInput).clear();
        cy.get(selectors.endDateInput).type('01/10/2034');

        cy.contains(endBeforeStartErrorText).should('exist');

        cy.get(selectors.applyButton).click();

        cy.get('@onSearch').should('not.have.been.called');
    });

    it('should not emit when the end date is empty', () => {
        setup();

        selectCondition('Between');

        cy.get(selectors.startDateInput).type('01/15/2034');
        cy.get(selectors.endDateInput).clear();

        cy.get(selectors.applyButton).click();

        cy.get('@onSearch').should('not.have.been.called');
    });

    it('should apply the "More than (days ago)" condition as >Nd format and clear the input', () => {
        setup();

        selectCondition('More than (days ago)');

        cy.get(selectors.relativeDaysInput).clear();
        cy.get(selectors.relativeDaysInput).type('365');
        cy.get(selectors.relativeApplyButton).click();

        cy.get('@onSearch').should('have.been.calledWithExactly', [
            {
                action: 'APPEND',
                category: 'CVE Created Time',
                value: '>365d',
            },
        ]);
        cy.get(selectors.relativeDaysInput).should('have.value', '');
    });

    it('should apply the "Between (days ago)" condition as Nd-Md format and clear the inputs', () => {
        setup();

        selectCondition('Between (days ago)');

        cy.get(selectors.relativeMinDaysInput).clear();
        cy.get(selectors.relativeMinDaysInput).type('30');
        cy.get(selectors.relativeMaxDaysInput).clear();
        cy.get(selectors.relativeMaxDaysInput).type('90');
        cy.get(selectors.relativeRangeApplyButton).click();

        cy.get('@onSearch').should('have.been.calledWithExactly', [
            {
                action: 'APPEND',
                category: 'CVE Created Time',
                value: '30d-90d',
            },
        ]);
        cy.get(selectors.relativeMinDaysInput).should('have.value', '');
        cy.get(selectors.relativeMaxDaysInput).should('have.value', '');
    });

    it('should not emit "More than (days ago)" when the input is empty', () => {
        setup();

        selectCondition('More than (days ago)');

        cy.get(selectors.relativeApplyButton).click();

        cy.get('@onSearch').should('not.have.been.called');
    });

    it('should not emit "Between (days ago)" when inputs are empty', () => {
        setup();

        selectCondition('Between (days ago)');

        cy.get(selectors.relativeRangeApplyButton).click();

        cy.get('@onSearch').should('not.have.been.called');
    });

    it('should not emit "Between (days ago)" when min exceeds max', () => {
        setup();

        selectCondition('Between (days ago)');

        cy.get(selectors.relativeMinDaysInput).clear();
        cy.get(selectors.relativeMinDaysInput).type('90');
        cy.get(selectors.relativeMaxDaysInput).clear();
        cy.get(selectors.relativeMaxDaysInput).type('30');
        cy.get(selectors.relativeRangeApplyButton).click();

        cy.get('@onSearch').should('not.have.been.called');
    });
});
