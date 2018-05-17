export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: 'tbody > tr',
    firstTableRow: 'tbody > tr:first',
    firstPanelTableRow: '.flex.flex-1 > table > tbody > tr:first',
    panelHeader: '.flex-row > .flex-1',
    clusterTableHeader: '.flex.flex-1 > table > thead > tr > th:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")'
};
