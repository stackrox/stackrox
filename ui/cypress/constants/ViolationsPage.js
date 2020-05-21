import { violationTagsSelectors } from '../selectors/tags';
import { violationCommentsSelectors, commentsDialogSelectors } from '../selectors/comments';
import selectSelectors from '../selectors/select';
import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/violations';

export const selectors = {
    navLink: 'nav li:contains("Violations") a',
    rows: '.rt-tbody .rt-tr',
    activeRow: '.row-active',
    firstTableRow: '.rt-tbody :nth-child(1) > .rt-tr',
    tableRowContains: (text) => `.rt-tbody .rt-tr:contains("${text}")`,
    firstPanelTableRow: '.rt-tbody > :nth-child(1) > .rt-tr',
    lastTableRow: '.rt-tr:last',
    panels: '[data-testid="panel"]',
    sidePanel: {
        header: '[data-testid="panel-header"]',
        tabs: 'button[data-testid="tab"]',
        getTabByIndex: (index) => `button[data-testid="tab"]:nth(${index})`,
        getTabByName: (name) => `button[data-testid="tab"]:contains("${name}")`,
        enforcementDetailMessage: '[data-testid="enforcement-detail-message"]',
        enforcementExplanationMessage: '[data-testid="enforcement-explanation-message"]',
        ...scopeSelectors('div[data-testid="panel"]:eq(1)', {
            enforcementTab: 'button[data-testid="tab"]:contains("Enforcement")',
            policyTab: 'button[data-testid="tab"]:contains("Policy")',
            getPropertyValue: (propertyName) =>
                ` div:not(:has(*)):contains("${propertyName}:"):first + *`,
            tags: violationTagsSelectors,
            comments: violationCommentsSelectors,
        }),
        closeButton: '[data-testid="cancel"]',
    },
    clusterTableHeader: '.rt-thead > .rt-tr > div:contains("Cluster")',
    viewDeploymentsButton: 'button:contains("View Deployments")',
    modal: '.ReactModalPortal > .ReactModal__Overlay',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    collapsible: {
        header: '.Collapsible__trigger',
        body: '.Collapsible__contentInner',
    },
    securityBestPractices: '[data-testid="deployment-security-practices"]',
    runtimeProcessCards: '[data-testid="runtime-processes"]',
    lifeCycleColumn: '.rt-thead.-header:contains("Lifecycle")',
    whitelistDeploymentButton: '[data-testid="whitelist-deployment-button"]',
    resolveButton: '[data-testid="resolve-button"]',
    whitelistDeploymentRow: '.rt-tr:contains("metadata-proxy-v0.1")',
    bulkAddTagsButton: '[data-testid="bulk-add-tags-button"]',
    addTagsDialog: scopeSelectors('.ReactModal__Content', {
        ...selectSelectors.multiSelect,
        confirmButton: 'button:contains("Confirm")',
        cancelButton: 'button:contains("Cancel")',
    }),
    commentsDialog: commentsDialogSelectors,
};
