export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: '.rt-tbody .rt-tr',
    firstTableRow: '.rt-tbody :nth-child(1) > .rt-tr',
    firstPanelTableRow: '.rt-tbody > :nth-child(1) > .rt-tr',
    lastTableRow: '.rt-tr:last',
    panels: '[data-test-id="panel"]',
    sidePanel: {
        header: '[data-test-id="panel-header"]',
        tabs: 'button.tab',
        getTabByIndex: index => `button.tab:nth(${index})`
    },
    clusterTableHeader: '.rt-thead > .rt-tr > div:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    collapsible: {
        header: '.Collapsible__trigger',
        body: '.Collapsible__contentInner'
    },
    securityBestPractices: '[data-test-id="deployment-security-practices"]',
    runtimeProcessCards: '.Collapsible'
};
