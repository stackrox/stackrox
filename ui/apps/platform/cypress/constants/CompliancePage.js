export const baseURL = '/main/compliance';

export const url = {
    dashboard: baseURL,
    entities: {
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        nodes: `${baseURL}/nodes`,
    },
    entity: {
        cluster: `${baseURL}/cluster`,
    },
    controls: `${baseURL}/controls`,
};

/*
 * Headings on entities pages.
 * The keys correspond to url entities object above.
 */
export const headingPlural = {
    clusters: 'Clusters',
    deployments: 'Deployments',
    namespaces: 'Namespaces',
    nodes: 'Nodes',
};

export const selectors = {
    scanButton: "[data-testid='scan-button']",
    export: {
        exportButton: "button:contains('Export')",
        pdfButton: "[data-testid='download-pdf-button']",
        csvButton: "[data-testid='download-csv-button']",
    },
    dashboard: {
        tileLinks: {
            cluster: {
                tile: "[data-testid='tile-link']:contains('cluster')",
                value: "[data-testid='tile-link']:contains('cluster') [data-testid='tile-link-value']",
            },
            namespace: {
                tile: "[data-testid='tile-link']:contains('namespace')",
                value: "[data-testid='tile-link']:contains('namespace') [data-testid='tile-link-value']",
            },
            node: {
                tile: "[data-testid='tile-link']:contains('node')",
                value: "[data-testid='tile-link']:contains('node') [data-testid='tile-link-value']",
            },
        },
    },
    list: {
        panels: '[data-testid="panel"]',
        sidePanelHeader: '[data-testid="panel-header"]:last',
        sidePanelCloseBtn: '[data-testid="panel"] .close-button',
        table: {
            header: '[data-testid="panel-header"]',
            firstGroup: '.table-group-active:first',
            firstTableGroup: '.rt-table:first',
            firstRow: 'div.rt-tr-group > .rt-tr.-odd:first',
            firstRowName: 'div.rt-tr-group > .rt-tr.-odd:first [data-testid="table-row-name"]',
            secondRow: 'div.rt-tr-group > .rt-tr.-even:first',
            secondRowName: 'div.rt-tr-group > .rt-tr.-even:first [data-testid="table-row-name"]',
            rows: "table tr:has('td')",
        },
    },
    widgets: "[data-testid='widget']",
    widget: {
        passingStandardsAcrossClusters: {
            widget: '[data-testid="standards-across-cluster"]',
            axisLinks: '[data-testid="standards-across-cluster"] a',
            barLabels: '[data-testid="standards-across-cluster"] .rv-xy-plot__series > text',
        },
        passingStandardsByCluster: {
            NISTBarLinks:
                '[data-testid="passing-standards-by-cluster"] g.vertical-cluster-bar-NIST rect',
        },
        passingStandardsAcrossNamespaces: {
            axisLinks: '[data-testid="standards-across-namespace"] a',
        },
        passingStandardsAcrossNodes: {
            axisLinks: '[data-testid="standards-across-node"] a',
        },
        controlsMostFailed: {
            widget: '[data-testid="link-list-widget"]:contains("failed")',
            listItems: '[data-testid="link-list-widget"]:contains("failed") a',
        },
        controlDetails: {
            widget: '[data-testid="control-details"]',
            standardName: '[data-testid="control-details"] [data-testid="standard-name"]',
            controlName: '[data-testid="control-details"] [data-testid="control-name"]',
        },
        PCICompliance: {
            controls:
                '[data-testid="PCI-compliance"] .widget-detail-bullet span:contains("Controls")',
        },
        relatedEntities: '[data-testid="related-resource-list"]',
    },
};
