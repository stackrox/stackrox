export const baseURL = '/main/compliance';

export const url = {
    dashboard: baseURL,
    list: {
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        nodes: `${baseURL}/nodes`,
        standards: {
            CIS_Docker_v1_2_0: `${baseURL}/controls?s[standard]=CIS%20Docker%20v1.2.0`,
            CIS_Kubernetes_v1_5: `${baseURL}/controls?s[standard]=CIS%20Kubernetes%20v1.5`,
            HIPAA_164: `${baseURL}/controls?s[standard]=HIPAA%20164`,
            NIST_800_190: `${baseURL}/controls/?s[standard]=NIST%20SP%20800-190`,
            PCI_DSS_3_2: `${baseURL}/controls?s[standard]=PCI%20DSS%203.2`
        }
    },
    entity: {
        cluster: `${baseURL}/cluster`
    }
};

export const selectors = {
    scanButton: "[data-testid='scan-button']",
    export: {
        exportButton: "button:contains('Export')",
        pdfButton: "[data-testid='download-pdf-button']",
        csvButton: "[data-testid='download-csv-button']"
    },
    dashboard: {
        tileLinks: {
            cluster: {
                tile: "[data-testid='tile-link']:contains('cluster')",
                value:
                    "[data-testid='tile-link']:contains('cluster') [data-testid='tile-link-value']"
            },
            namespace: {
                tile: "[data-testid='tile-link']:contains('namespace')",
                value:
                    "[data-testid='tile-link']:contains('namespace') [data-testid='tile-link-value']"
            },
            node: {
                tile: "[data-testid='tile-link']:contains('node')",
                value: "[data-testid='tile-link']:contains('node') [data-testid='tile-link-value']"
            }
        }
    },
    list: {
        panels: '[data-testid="panel"]',
        sidePanelHeader: '[data-testid="panel-header"]:last',
        sidePanelCloseBtn: '[data-testid="panel"] .close-button',
        banner: {
            content: '[data-testid="collapsible-banner"]',
            collapseButton: '[data-testid="banner-collapse-button"]'
        },
        table: {
            header: '[data-testid="panel-header"]',
            firstGroup: '.table-group-active:first',
            firstTableGroup: '.rt-table:first',
            firstRow: 'div.rt-tr-group > .rt-tr.-odd:first',
            firstRowName: 'div.rt-tr-group > .rt-tr.-odd:first [data-testid="table-row-name"]',
            secondRow: 'div.rt-tr-group > .rt-tr.-even:first',
            secondRowName: 'div.rt-tr-group > .rt-tr.-even:first [data-testid="table-row-name"]',
            rows: "table tr:has('td')"
        }
    },
    widgets: "[data-testid='widget']",
    widget: {
        controlsInCompliance: {
            widget: '[data-testid="compliance-across-entities"]',
            centerLabel:
                '[data-testid="compliance-across-entities"] svg .rv-xy-plot__series--label text',
            passingControls:
                '[data-testid="compliance-across-entities"] [data-testid="passing-controls-value"]',
            failingControls:
                '[data-testid="compliance-across-entities"] [data-testid="failing-controls-value"]',
            arcs: '[data-testid="compliance-across-entities"] svg path'
        },
        passingStandardsAcrossClusters: {
            widget: '[data-testid="standards-across-cluster"]',
            axisLinks: '[data-testid="standards-across-cluster"] a',
            barLabels: '[data-testid="standards-across-cluster"] .rv-xy-plot__series > text'
        },
        passingStandardsByCluster: {
            NISTBarLinks:
                '[data-testid="passing-standards-by-cluster"] g.vertical-cluster-bar-NIST rect'
        },
        passingStandardsAcrossNamespaces: {
            axisLinks: '[data-testid="standards-across-namespace"] a'
        },
        passingStandardsAcrossNodes: {
            axisLinks: '[data-testid="standards-across-node"] a'
        },
        controlsMostFailed: {
            widget: '[data-testid="link-list-widget"]:contains("failed")',
            listItems: '[data-testid="link-list-widget"]:contains("failed") a'
        },
        controlDetails: {
            widget: '[data-testid="control-details"]',
            standardName: '[data-testid="control-details"] [data-testid="standard-name"]',
            controlName: '[data-testid="control-details"] [data-testid="control-name"]'
        },
        PCICompliance: {
            controls:
                '[data-testid="PCI-compliance"] .widget-detail-bullet span:contains("Controls")'
        },
        relatedEntities: '[data-testid="related-resource-list"]'
    }
};
