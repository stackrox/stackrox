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
            return Promise.resolve();
        });
}
