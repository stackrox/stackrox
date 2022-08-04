import navigationSelectors from '../selectors/navigation';

export const url = '/main/violations';

export const selectors = {
    navLink: `${navigationSelectors.navLinks}:contains("Violations")`,
    tableRow: 'tbody tr',
    firstTableRow: 'tbody tr:nth(0)',
    tableRowContains: (text) => `tbody tr:contains("${text}")`,
    firstTableRowLink: 'tbody tr:nth(0) a',
    lastTableRow: 'tbody tr:last',
    lastTableRowLink: 'tbody tr:last a',
    resultsFoundHeader: (number) =>
        `h2:contains("${number} result${number === 1 ? '' : 's'} found")`,
    actions: {
        btn: 'td.pf-c-table__action button',
        excludeDeploymentBtn: 'button:contains("Exclude deployment")',
        resolveBtn: 'button:contains("Mark as resolved")',
        resolveAndAddToBaselineBtn: 'button:contains("Resolve and add to process baseline")',
        dropdown: '[data-testid="violations-bulk-actions-dropdown"]',
        addTagsBtn: '[data-testid="bulk-add-tags-btn"]',
    },
    details: {
        page: '[data-testid="violation-details-page"]',
        title: 'h1.pf-c-title',
        subtitle: 'h2.pf-c-title',
        tabs: 'li.pf-c-tabs__item',
        violationTab: 'li.pf-c-tabs__item:contains("Violation")',
        enforcementTab: 'li.pf-c-tabs__item:contains("Enforcement")',
        deploymentTab: 'li.pf-c-tabs__item:contains("Deployment")',
        policyTab: 'li.pf-c-tabs__item:contains("Policy")',
        tags: {
            input: '[data-testid="violation-tags"] input',
            values: '[data-testid="violation-tags"] .pf-c-chip-group div.pf-c-chip',
            removeValueButton: (tag) =>
                `[data-testid="violation-tags"] div.pf-c-chip:contains(${tag}) button`,
            clearAllTagsButton: '[data-testid="violation-tags"] .pf-c-select__toggle-clear',
        },
    },
    enforcement: {
        detailMessage: '[data-testid="enforcement-detail-message"]',
        explanationMessage: '[data-testid="enforcement-explanation-message"]',
    },
    deployment: {
        overview: '[data-testid="deployment-details"] [data-testid="deployment-overview"]',
        containerConfiguration:
            '[data-testid="deployment-details"] [data-testid="container-configuration"]',
        securityContext: '[data-testid="deployment-details"] [data-testid="security-context"]',
        portConfiguration: '[data-testid="deployment-details"] [data-testid="port-configuration"]',
        snapshotWarning:
            '[data-testid="deployment-details"] [data-testid="deployment-snapshot-warning"]',
    },
    table: {
        rows: 'tbody tr',
    },
    modal: {
        excludeConfirmation: '[data-testid="exclude-confirmation-modal"]',
        tagConfirmation: {
            input: '[data-testid="tag-confirmation-modal"] input',
            options: '[data-testid="tag-confirmation-modal"] li button',
            cancelBtn: 'footer button:contains("Cancel")',
            confirmBtn: 'footer button:contains("Confirm")',
        },
        resolveConfirmation: '[data-testid="resolve-confirmation-modal"]',
    },
    viewDeploymentsButton: 'button:contains("View Deployments")',
    clusterFieldInModal: '.ReactModalPortal > .ReactModal__Overlay span:contains("Cluster")',
    securityBestPractices: '[data-testid="deployment-security-practices"]',
    runtimeProcessCards: '[data-testid="runtime-processes"]',
    excludedDeploymentRow: '.rt-tr:contains("metadata-proxy-v0.1")',
};
