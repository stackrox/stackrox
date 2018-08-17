export const url = '/main/risk';

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    panelTabs: {
        riskIndicators: 'button.tab:contains("Risk Indicators")',
        deploymentDetails: 'button.tab:contains("Deployment Details")'
    },
    cancelButton: 'button[data-test-id="cancel"]',
    search: {
        searchModifier: '.risk-search-input #react-select-3--value > :nth-child(1)',
        searchWord: '.risk-search-input #react-select-3--value > :nth-child(2)'
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
            prevent_sensor: 'div div.cursor-pointer:contains("sensor")',
            firstRow: 'div.rt-tr:first-child'
        }
    }
};
