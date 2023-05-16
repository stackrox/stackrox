export const selectors = {
    list: {
        table: {
            firstGroup: '.table-group-active:first',
            firstTableGroup: '.rt-table:first',
            firstRow: 'div.rt-tr-group > .rt-tr.-odd:first',
            firstRowName: 'div.rt-tr-group > .rt-tr.-odd:first [data-testid="table-row-name"]',
            secondRowName: 'div.rt-tr-group > .rt-tr.-even:first [data-testid="table-row-name"]',
        },
    },
    widget: {
        passingStandardsAcrossClusters: {
            axisLinks: '[data-testid="standards-across-cluster"] a',
        },
        passingStandardsAcrossNamespaces: {
            axisLinks: '[data-testid="standards-across-namespace"] a',
        },
        passingStandardsAcrossNodes: {
            axisLinks: '[data-testid="standards-across-node"] a',
        },
        PCICompliance: {
            controls:
                '[data-testid="PCI-compliance"] .widget-detail-bullet span:contains("Controls")',
        },
        relatedEntities: '[data-testid="related-resource-list"]',
    },
};
