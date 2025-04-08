export const selectors = {
    list: {
        table: {
            firstGroup: '.table-group-active:first',
            firstTableGroup: '.rt-table:first',
            firstRow: 'div.rt-tr-group > .rt-tr.-odd:first',
            firstRowName: 'div.rt-tr-group > .rt-tr.-odd:first [data-testid="table-row-name"]',
            secondRowName: 'div.rt-tr-group > .rt-tr.-even:first [data-testid="table-row-name"]',
            firstStandard: 'div.rt-table .rt-thead .rt-th:nth-child(3)',
            firstPercentage: 'div.rt-tr-group > .rt-tr.-odd > .rt-td:nth-child(3)',
        },
    },
};
