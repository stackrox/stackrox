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
        cves: `${baseURL}/cves`
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
    viewAllButton: 'button:contains("View All")'
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
