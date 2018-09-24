export const url = '/main/compliance';

export const selectors = {
    compliance: 'nav.left-navigation li:contains("Compliance") a',
    firstNavLink: '.navigation-panel li:nth-child(2) a',
    secondNavLink: '.navigation-panel li:nth-child(3) a',
    benchmarkTabs: 'button.tab',
    scanNowButton: 'button.rounded-sm.bg-success-500',
    checkRows: 'div.rt-tbody div.rt-tr-group',
    passColumns: 'div.rt-tbody div.rt-tr-group:first-child .rt-tr .rt-td:nth-child(3)',
    hostColumns: '[data-test-id="panel"] div.rt-tbody div.rt-tr-group:not(.hidden)',
    select: {
        day: 'select:first',
        time: 'select:last'
    },
    clusterList: '.navigation-panel ul > li',
    leftNavigation: '.left-navigation'
};
