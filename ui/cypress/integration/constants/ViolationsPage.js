export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: '.rt-tbody .rt-tr',
    firstTableRow: '.rt-tbody :nth-child(1) > .rt-tr',
    firstPanelTableRow: '.rt-tbody > :nth-child(1) > .rt-tr',
    lastTableRow: ':nth-child(4) > .rt-tr',
    sidePanel: {
        panel: 'div[data-test-id="panel"]',
        header: 'div[data-test-id="panel"] .flex-row > .flex-1',
        tabs: 'div[data-test-id="panel"] button.tab',
        getTabByIndex: index => `div[data-test-id="panel"] button.tab:nth(${index})`
    },
    clusterTableHeader: '.rt-thead > .rt-tr > div:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    collapsible: {
        header: '.Collapsible__trigger h1',
        body: '.Collapsible__contentInner'
    },
    securityBestPractices: '[data-test-id="deployment-security-practices"]'
};
