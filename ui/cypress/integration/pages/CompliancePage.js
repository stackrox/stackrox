export const url = '/main/compliance';

export const selectors = {
    compliance: 'nav.left-navigation li:contains("Compliance") a',
    navLink: '.navigation-panel li:nth-child(2) a',
    benchmarkTabs: 'button.tab',
    scanNowButton: 'button.rounded-sm.bg-success-500',
    checkRows: 'div.overflow-y-scroll table tbody tr',
    passColumns: 'div.overflow-y-scroll table tbody tr td:nth-child(3)',
    hostColumns: '.border-t > .flex-col tbody tr',
    select: {
        day: 'select:first',
        time: 'select:last'
    }
};
