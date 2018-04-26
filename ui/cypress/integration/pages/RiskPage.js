export const url = '/main/risk';

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    panelTabs: 'button.tab',
    cancelButton: 'button.cancel',
    search: {
        searchModifier: '.risk-search-input #react-select-3--value > :nth-child(1)',
        searchWord: '.risk-search-input #react-select-3--value > :nth-child(2)'
    }
};
