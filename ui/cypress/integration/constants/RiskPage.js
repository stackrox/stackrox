export const url = '/main/risk';

export const errorMessages = {
    deploymentNotFound: 'Deployment not found',
    riskNotFound: 'Risk not found',
    processNotFound: 'No processes discovered'
};

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    errMgBox: 'div.error-message',
    panelTabs: {
        riskIndicators: 'button[data-test-id="tab"]:contains("Risk Indicators")',
        deploymentDetails: 'button[data-test-id="tab"]:contains("Deployment Details")',
        processDiscovery: 'button[data-test-id="tab"]:contains("Process Discovery")'
    },
    cancelButton: 'button[data-test-id="cancel"]',
    search: {
        searchModifier: '.react-select__multi-value__label:first',
        searchWord: '.react-select__multi-value__label:eq(1)'
    },
    mounts: {
        label: 'div:contains("Mounts"):last',
        items: 'div:contains("Mounts"):last + ul li div'
    },
    imageLink: 'div:contains("Image Name") + a',
    table: {
        columns: {
            priority: 'div.rt-th div:contains("Priority")'
        },
        row: {
            firstRow: 'div.rt-tr-group:first-child div.rt-tr'
        }
    },
    networkNodeLink: '[data-test-id="network-node-link"]'
};
