export const url = '/main/risk';

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    panelTabs: {
        riskIndicators: 'button.tab:contains("Risk Indicators")',
        deploymentDetails: 'button.tab:contains("Deployment Details")'
    },
    cancelButton: 'button.cancel',
    search: {
        searchModifier: '.risk-search-input #react-select-3--value > :nth-child(1)',
        searchWord: '.risk-search-input #react-select-3--value > :nth-child(2)'
    },
    mounts: {
        label: 'div:contains("Mounts")',
        items: 'div:contains("Mounts") + ul > li'
    },
    imageLink: 'div:contains("Image Name") + a'
};
