export const url = '/main/risk';

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    panelTabs: {
        riskIndicators: 'button.tab:contains("Risk Indicators")',
        deploymentDetails: 'button.tab:contains("Deployment Details")'
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
