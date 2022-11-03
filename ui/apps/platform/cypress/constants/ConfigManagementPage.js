import scopeSelectors from '../helpers/scopeSelectors';
import tableSelectors from '../selectors/table';

export const dashboardSelectors = {
    widgets: "[data-testid='widget']",
    tileLinks: "[data-testid='tile-link']",
    tileLinkValue: "[data-testid='tile-link-value']",
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    rbacVisibilityDropdown: 'button:contains("RBAC")',
    getMenuListItem: (name) => {
        return `[data-testid="menu-list"] [data-testid="${name}"]`;
    },
    getWidget: (title) => {
        return `[data-testid="widget"]:contains('${title}')`;
    },
    viewAllButton: 'button:contains("View All")',
    viewStandardButton: 'button:contains("View Standard")',
    policyViolationsBySeverity: {
        link: {
            ratedAsHigh:
                '[data-testid="widget"]:contains("Policy Violations by Severity") a:contains("rated as high")',
            ratedAsLow:
                '[data-testid="widget"]:contains("Policy Violations by Severity") a:contains("rated as low")',
            policiesWithoutViolations:
                '[data-testid="widget"]:contains("Policy Violations by Severity") a:contains("policies")',
        },
    },
    cisStandardsAcrossClusters: {
        widget: '[data-testid="compliance-by-controls"]',
        select: {
            input: '[data-testid="compliance-by-controls"] .react-select__control',
            value: '[data-testid="compliance-by-controls"] .react-select__single-value',
            options: '[data-testid="compliance-by-controls"] .react-select__option',
        },
        passingControlsLink: 'a[title*="Controls Passing"]',
        failingControlsLinks: 'a[title*="Controls Failing"]',
    },
    horizontalBars: '.rv-xy-plot__series.rv-xy-plot__series--bar > rect',
};

export const listSelectors = {
    disabledTableRows: '.rt-tr-group > .data-test-disabled',
    tableRows: '.rt-tr-group > .rt-tr',
    tableCells: '.rt-td',
    tableLinks: '.rt-tbody .rt-td a',
    tablePanelHeader: '[data-testid="panel"] [data-testid="panel-header"]',
    tableNextPage: '[data-testid="next-page-button"]',
    sidePanel: '[data-testid="side-panel"]',
};

export const entitySelectors = {
    metadataWidget: '[data-testid="widget"]:contains("Metadata")',
    externalLink: '[data-testid="side-panel"] [data-testid="external-link"]',
    countWidgets: '[data-testid="related-entity-list-count"]',
    countWidgetTitle: '[data-testid="related-entity-list-count-title"]',
    countWidgetValue: '[data-testid="related-entity-list-count-value"]',
    relatedEntityWidgets: '[data-testid="related-entity"]',
    relatedEntityWidgetTitle: '[data-testid="related-entity-title"]',
    relatedEntityWidgetValue: '[data-testid="related-entity-value"]',
    groupedTabs: '[data-testid="grouped-tab"] [data-testid="tab"]',
    failingNodes: '[data-testid="widget"] .rt-tr-group > .rt-tr',
    deploymentsWithFailedPolicies: '[data-testid="deployments-with-failed-policies"]',
};

const sidePanelSelectors = {
    policyFindingsSection: scopeSelectors('[data-testid="policy-findings-section"]', {
        table: tableSelectors,
    }),
    violationTags: {
        input: '[data-testid="violation-tags"] input',
        values: '[data-testid="violation-tags"] .pf-c-chip-group div.pf-c-chip',
        removeValueButton: (tag) =>
            `[data-testid="violation-tags"] div.pf-c-chip:contains(${tag}) button`,
    },
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...entitySelectors,
    // TODO-ivan: do the proper scoped definitions for the above
    mainTable: scopeSelectors('[data-testid="panel"]', tableSelectors),
    sidePanel1: scopeSelectors('[data-testid="side-panel"]', sidePanelSelectors),
};
