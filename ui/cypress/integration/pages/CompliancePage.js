export const url = '/main/compliance';

export const selectors = {
    navLink: 'nav li:contains("Compliance") a',
    benchmarkTabs: 'button.tab',
    scanNowButton: 'button.rounded-sm.bg-success-500',
    checkRows: 'div.overflow-y-scroll table tbody tr',
    passColumns: 'div.overflow-y-scroll table tbody tr td:nth-child(3)',
    hostColumns: '.border-t > .flex-col tbody tr'
};
