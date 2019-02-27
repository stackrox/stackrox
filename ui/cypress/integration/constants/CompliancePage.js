export const baseURL = '/main/compliance';

export const url = {
    dashboard: baseURL,
    list: {
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        nodes: `${baseURL}/nodes`,
        standards: {
            CIS_Docker_v1_1_0: `${baseURL}/CIS_Docker_v1_1_0`,
            CIS_Kubernetes_v1_2_0: `${baseURL}/CIS_Kubernetes_v1_2_0`,
            HIPAA_164: `${baseURL}/HIPAA_164`,
            NIST_800_190: `${baseURL}/NIST_800_190`,
            PCI_DSS_3_2: `${baseURL}/PCI_DSS_3_2`
        }
    }
};

const getWidget = text =>
    `[data-test-id='widget']:has([data-test-id='widget-header']:contains("${text}"))`;
const getControlsInCompliance = getWidget('Controls in Compliance');
const getPassingStandardsAcrossClusters = getWidget('Passing standards across CLUSTERs');
const getControlsMostFailed = getWidget('Controls most failed');
const getControlDetails = getWidget('Control details');

export const selectors = {
    scanButton: "button:contains('Scan environment')",
    export: {
        exportButton: "button:contains('Export')",
        pdfButton: "[data-test-id='download-pdf-button']",
        csvButton: "[data-test-id='download-csv-button']"
    },
    dashboard: {
        tileLinks: {
            cluster: {
                tile: "[data-test-id='tile-link']:contains('cluster')",
                value:
                    "[data-test-id='tile-link']:contains('cluster') [data-test-id='tile-link-value']"
            },
            namespace: {
                tile: "[data-test-id='tile-link']:contains('namespace')",
                value:
                    "[data-test-id='tile-link']:contains('namespace') [data-test-id='tile-link-value']"
            },
            node: {
                tile: "[data-test-id='tile-link']:contains('node')",
                value:
                    "[data-test-id='tile-link']:contains('node') [data-test-id='tile-link-value']"
            }
        }
    },
    list: {
        tableRows: "table tr:has('td')"
    },
    widgets: "[data-test-id='widget']",
    widget: {
        controlsInCompliance: {
            widget: getControlsInCompliance,
            centerLabel: `${getControlsInCompliance} svg .rv-xy-plot__series--label text`,
            passingControls: `${getControlsInCompliance} [data-test-id='passing-controls-value']`,
            failingControls: `${getControlsInCompliance} [data-test-id='failing-controls-value']`,
            arcs: `${getControlsInCompliance} svg path`
        },
        passingStandardsAcrossClusters: {
            widget: getPassingStandardsAcrossClusters,
            axisLinks: `${getPassingStandardsAcrossClusters} a`,
            barLabels: `${getPassingStandardsAcrossClusters} svg .rv-xy-plot__series text`
        },
        controlsMostFailed: {
            widget: getControlsMostFailed,
            listItems: `${getControlsMostFailed} a`
        },
        controlDetails: {
            widget: getControlDetails,
            standardName: `${getControlDetails} [data-test-id='standard-name']`,
            controlname: `${getControlDetails} [data-test-id='control-name']`
        }
    }
};
