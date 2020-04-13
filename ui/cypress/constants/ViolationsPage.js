export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: '.rt-tbody .rt-tr',
    firstTableRow: '.rt-tbody :nth-child(1) > .rt-tr',
    firstPanelTableRow: '.rt-tbody > :nth-child(1) > .rt-tr',
    lastTableRow: '.rt-tr:last',
    panels: '[data-testid="panel"]',
    sidePanel: {
        header: '[data-testid="panel-header"]',
        tabs: 'button[data-testid="tab"]',
        getTabByIndex: index => `button[data-testid="tab"]:nth(${index})`,
        enforcementDetailMessage: '[data-testid="enforcement-detail-message"]',
        enforcementExplanationMessage: '[data-testid="enforcement-explanation-message"]'
    },
    clusterTableHeader: '.rt-thead > .rt-tr > div:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    collapsible: {
        header: '.Collapsible__trigger',
        body: '.Collapsible__contentInner'
    },
    securityBestPractices: '[data-testid="deployment-security-practices"]',
    runtimeProcessCards: '[data-testid="runtime-processes"]',
    lifeCycleColumn: '.rt-thead.-header:contains("Lifecycle")',
    whitelistDeploymentButton: '[data-testid="whitelist-deployment-button"]',
    resolveButton: '[data-testid="resolve-button"]',
    whitelistDeploymentRow: '.rt-tr:contains("metadata-proxy-v0.1")'
};
