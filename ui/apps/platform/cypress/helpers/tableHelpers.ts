export function getTableRowLinkByName(name: string) {
    const exactName = new RegExp(`^${name}$`, 'g');
    return cy
        .get('a')
        .contains(exactName)
        .then(($el) => {
            return cy.wrap($el);
        });
}

export function getTableRowActionButtonByName(name: string) {
    const exactName = new RegExp(`^${name}$`, 'g');
    return cy
        .get('a')
        .contains(exactName)
        .then(($el) => {
            return cy.wrap($el).parent().siblings('td').find('button[aria-label="Kebab toggle"]');
        });
}

export function openTableRowActionMenu(
    rowSelector: string,
    menuButtonAriaLabel: string = 'Kebab toggle'
) {
    return cy.get(`${rowSelector} button[aria-label="${menuButtonAriaLabel}"]`).click();
}

export function editIntegration(name: string) {
    cy.get(`tr:contains('${name}') td.pf-v5-c-table__action button`).click();
    cy.get(
        `tr:contains('${name}') td.pf-v5-c-table__action button:contains("Edit Integration")`
    ).click();
}

export function queryTableHeader(headerName: string) {
    return cy.get(`th`).contains(new RegExp(`^${headerName}$`));
}

export function queryTableSortHeader(headerName: string) {
    return cy
        .get(`th button:has('.pf-v5-c-table__sort-indicator')`)
        .contains(new RegExp(`^${headerName}$`));
}

export function sortByTableHeader(headerName: string) {
    return queryTableSortHeader(headerName).click();
}

export function changePerPageOption(amount: number) {
    cy.get('button').contains(new RegExp('\\d+ - \\d+ of \\d+')).click();
    cy.get('button[role="menuitem"]').contains(`${amount} per page`).click();
}

export function paginateNext() {
    return cy.get('button[aria-label="Go to next page"]').click();
}

export function paginatePrevious() {
    return cy.get('button[aria-label="Go to previous page"]').click();
}

export function assertOnEachRowForColumn(
    columnDataLabel: string,
    // eslint-disable-next-line no-unused-vars
    assertion: (index: number, el: HTMLElement) => void
) {
    return cy
        .get(`table:has(th:contains("${columnDataLabel}"))`)
        .then(($el) => $el.find(`td[data-label="${columnDataLabel}"]`))
        .then(($cells) => {
            if ($cells.length === 0) {
                cy.log(
                    `No rows found for the column [${columnDataLabel}], assertion will pass applied to 0 rows`
                );
            }
            $cells.each(assertion);
        });
}

// Opens the column management modal, resets the columns to default, and saves the changes
function resetManagedColumns() {
    cy.get('button:contains("Columns")').click();
    cy.get('button:contains("Reset to default")').click();
    cy.get('button:contains("Save")').click();
}

/**
 * Verifies the functionality of the column management modal
 *
 * This helper opens the column management modal, and for every listed column, hides the column and asserts
 * that the column is hidden. This makes the assumption that each label in the modal corresponds to a column
 * header in the table.
 *
 * @param { tableSelector } tableSelector - The selector for the table to manage columns for
 */
export function verifyColumnManagement({ tableSelector }: { tableSelector: string }) {
    resetManagedColumns();

    // Open the colum management modal and get the list of columns
    cy.get('button:contains("Columns")').click();
    cy.get('.pf-v5-c-modal-box label')
        .then(($labels) => {
            cy.get('.pf-v5-c-modal-box button:contains("Cancel")').click();
            const columns = $labels.map((_, el) => el.innerText).get();
            return cy.wrap(columns);
        })
        .then((columns) => {
            // For each column, hide the column and assert that the column is hidden
            columns.forEach((column) => {
                const columnLabel = new RegExp(`^${column}$`, 'g');
                // Assert that the table has a header for the column and that it is visible
                cy.get(tableSelector).contains('th', columnLabel).should('be.visible');

                // Hide the column
                cy.get('button:contains("Columns")').click();
                cy.get('label').contains(columnLabel).click();
                cy.get('button:contains("Save")').click();

                // Assert that the table header for the column is hidden
                cy.get(tableSelector).contains('th', columnLabel).should('not.be.visible');
            });
        });

    resetManagedColumns();
}

export function assertVisibleTableColumns(tableSelector: string, columns: string[]) {
    cy.get(`${tableSelector} > thead th:not(.pf-v5-u-display-none)`).should(
        'have.length',
        columns.length
    );
    columns.forEach((column) => {
        cy.get(`${tableSelector} > thead th:not(.pf-v5-u-display-none)`).should(
            'contain.text',
            column
        );
    });
}
