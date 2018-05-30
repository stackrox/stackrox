export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: 'tbody > tr',
    firstTableRow: 'tbody > tr:first',
    firstPanelTableRow: 'table > tbody > tr:first',
    sidePanel: {
        panel: '.side-panel',
        header: '.side-panel .flex-row > .flex-1',
        tabs: '.side-panel button.tab',
        getTabByIndex: index => `.side-panel button.tab:nth(${index})`
    },
    clusterTableHeader: 'table thead:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    collapsible: {
        header: '.Collapsible__trigger h1',
        body: '.Collapsible__contentInner'
    }
};
