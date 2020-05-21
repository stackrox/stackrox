const table = {
    header: '[data-testid="filtered-header"]',
    column: {
        name: 'div.rt-th div:contains("Name")',
        priority: 'div.rt-th div:contains("Priority")',
    },
    row: {
        firstRow: 'div.rt-tr-group:first-child div.rt-tr',
    },
    rows: 'div.rt-tr-group div.rt-tr',
    cells: '.rt-td',
    columnHeaders: 'div.rt-th',
    activeRow: 'div.rt-tr-group .row-active',
    /** @deprecated use 'cells' instead as it better reflects the nature of this selector */
    columns: '.rt-td',
    dataRows: '.rt-tbody .rt-tr-group .rt-tr',
};

export default table;
