export const baseURL = '/main/configmanagement';

export const url = {
    dashboard: baseURL,
    list: {
        policies: `${baseURL}/policies`,
        controls: `${baseURL}/controls`,
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        nodes: `${baseURL}/nodes`,
        deployments: `${baseURL}/deployments`,
        images: `${baseURL}/images`,
        secrets: `${baseURL}/secrets`,
        subjects: `${baseURL}/subjects`,
        serviceAccounts: `${baseURL}/serviceaccounts`,
        roles: `${baseURL}/roles`
    },
    single: {
        policy: `${baseURL}/policy`,
        control: `${baseURL}/control`,
        cluster: `${baseURL}/cluster`,
        namespace: `${baseURL}/namespace`,
        node: `${baseURL}/node`,
        deployment: `${baseURL}/deployment`,
        image: `${baseURL}/image`,
        secret: `${baseURL}/secret`,
        subject: `${baseURL}/subject`,
        serviceAccount: `${baseURL}/serviceaccount`,
        role: `${baseURL}/role`
    }
};

export const controlStatus = {
    pass: 'Pass',
    fail: 'Fail'
};

export const dashboardSelectors = {
    widgets: "[data-test-id='widget']",
    tileLinks: "[data-test-id='tile-link']",
    tileLinkValue: "[data-test-id='tile-link-value']",
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    rbacVisibilityDropdown: 'button:contains("RBAC")',
    getMenuListItem: name => {
        return `[data-test-id="menu-list"] [data-test-id="${name}"]`;
    },
    getWidget: title => {
        return `[data-test-id="widget"]:contains('${title}')`;
    },
    viewAllButton: 'button:contains("View All")',
    viewStandardButton: 'button:contains("View Standard")',
    policyViolationsBySeverity: {
        link: {
            ratedAsHigh:
                '[data-test-id="widget"]:contains("Policy Violations by Severity") a:contains("rated as high")',
            ratedAsLow:
                '[data-test-id="widget"]:contains("Policy Violations by Severity") a:contains("rated as low")',
            policiesWithoutViolations:
                '[data-test-id="widget"]:contains("Policy Violations by Severity") a:contains("policies")'
        }
    },
    cisStandardsAcrossClusters: {
        widget: '[data-test-id="compliance-by-controls"]',
        select: {
            input: '[data-test-id="compliance-by-controls"] .react-select__control',
            value: '[data-test-id="compliance-by-controls"] .react-select__single-value',
            options: '[data-test-id="compliance-by-controls"] .react-select__option'
        },
        passingControlsLink: 'a:contains("Controls Passing")',
        failingControlsLinks: 'a:contains("Controls Failing")'
    },
    horizontalBars: '.rv-xy-plot__series.rv-xy-plot__series--bar > rect'
};

export const listSelectors = {
    disabledTableRows: '.rt-tr-group > .data-test-disabled',
    tableRows: '.rt-tr-group > .rt-tr',
    tableCells: '.rt-td',
    tableLinks: '.rt-tr-group > .rt-tr > .rt-td > a',
    tablePanelHeader: '[data-test-id="panel"] [data-test-id="panel-header"]',
    tableNextPage: '[data-test-id="next-page-button"]'
};

export const entitySelectors = {
    metadataWidget: '[data-test-id="widget"]:contains("Metadata")',
    externalLink: '[data-test-id="side-panel"] [data-test-id="external-link"]',
    countWidgets: '[data-test-id="related-entity-list-count"]',
    countWidgetTitle: '[data-test-id="related-entity-list-count-title"]',
    countWidgetValue: '[data-test-id="related-entity-list-count-value"]',
    relatedEntityWidgets: '[data-test-id="related-entity"]',
    relatedEntityWidgetTitle: '[data-test-id="related-entity-title"]',
    relatedEntityWidgetValue: '[data-test-id="related-entity-value"]',
    groupedTabs: '[data-test-id="grouped-tab"] [data-test-id="tab"]',
    failingNodes: '[data-test-id="widget"] .rt-tr-group > .rt-tr'
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...entitySelectors
};
