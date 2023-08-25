export const dashboardSelectors = {
    tileLinks: "[data-testid='tile-link']",
    tileLinkValue: "[data-testid='tile-link-value']",
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    rbacVisibilityDropdown: 'button:contains("Role-Based Access Control")',
    getMenuListItem: (name) => {
        return `[data-testid="menu-list"] [data-testid="${name}"]`;
    },
    getWidget: (title) => {
        return `[data-testid="widget"]:contains('${title}')`;
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
    sidePanel: '[data-testid="side-panel"]',
};

export const entitySelectors = {
    countWidgets: '[data-testid="related-entity-list-count"]',
    countWidgetTitle: '[data-testid="related-entity-list-count-title"]',
    countWidgetValue: '[data-testid="related-entity-list-count-value"]',
    relatedEntityWidgets: '[data-testid="related-entity"]',
    relatedEntityWidgetTitle: '[data-testid="related-entity-title"]',
    groupedTabs: '[data-testid="grouped-tab"] [data-testid="tab"]',
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...entitySelectors,
};
