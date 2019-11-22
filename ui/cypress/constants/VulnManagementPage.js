export const baseURL = '/main/vulnerability-management';

export const url = {
    dashboard: baseURL,
    list: {
        policies: `${baseURL}/policies`,
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        deployments: `${baseURL}/deployments`,
        images: `${baseURL}/images`,
        components: `${baseURL}/components`,
        cves: `${baseURL}/cves`,
        image: `${baseURL}/image`
    }
};

export const listSelectors = {
    riskScoreCol: '.rt-table > .rt-tbody > div > div > div:nth-child(10)',
    componentsRiskScoreCol: '.rt-table > .rt-tbody >div > div > div:nth-child(7)',
    cvesCvssScoreCol: '.rt-table > .rt-tbody > div > .rt-tr.-odd > div:nth-child(4) > div > span',
    tableRows: '.rt-tr',
    tableColumn: '.rt-th.leading-normal > div',
    tableColumnLinks: '.rt-tr-group > .rt-tr > .rt-td > a',
    tableCVEColumnLinks: '.rt-tr-group > .rt-tr > .rt-td > .items-center'
};

export const dashboardSelectors = {
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    topRiskyItems: {
        select: {
            input: '[data-test-id="widget"] .react-select__control',
            value: '[data-test-id="widget"] .react-select__single-value',
            options: '[data-test-id="widget"] .react-select__option'
        }
    },
    getMenuListItem: name => {
        return `[data-test-id="menu-list"] [data-test-id="${name}"]`;
    },
    getWidget: title => {
        return `[data-test-id="widget"]:contains('${title}')`;
    },
    getTileLink: title => {
        return `[data-test-id="tile-link"]:contains('${title}')`;
    },
    getAllClickableTileLinks: title => {
        return `#capture-dashboard-stretch > div > .h-full > div > ul > li > a > div > div > div:contains('${title}')`;
    },
    viewAllButton: 'button:contains("View All")',
    dataRowLink: '[data-testid="numbered-list-item-name"]',
    entityPageHeader: '[data-test-id="header-text"]',
    tabLinks: '[data-test-id="tab"]',
    allTileLinks: '#capture-dashboard-stretch > div > .h-full > div > ul > li',
    tabHeader: '[data-test-id="panel-header"]'
};

const linkSelectors = {
    allCvesLink: '[data-testid="allCvesLink"]',
    fixableCvesLink: '[data-testid="fixableCvesLink"]',
    tileLinks: "[data-test-id='tile-link']",
    tileLinkValue: "[data-test-id='tile-link-value']",
    tileLinkSuperText: '[data-testid="tileLinkSuperText"]'
};

const sidepanelSelectors = {
    backButton: '[data-testid="sidepanelBackButton"]'
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...linkSelectors,
    ...sidepanelSelectors
};
