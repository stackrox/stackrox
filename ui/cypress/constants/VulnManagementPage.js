import { selectors as tablePaginationSelectors } from './TablePagination';
import sidePanelSelectors from '../selectors/panel';

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
        image: `${baseURL}/image`,
        cve: `${baseURL}/cve`,
        policy: `${baseURL}/policy`,
        deployment: `${baseURL}/deployment`
    }
};

export const vmHomePageSelectors = {
    vmDBPageTileLink: '[data-test-id="Vulnerability Management"]'
};
export const listSelectors = {
    riskScoreCol: '.rt-table > .rt-tbody > div > div > div:nth-child(10)',
    componentsRiskScoreCol: '.rt-table > .rt-tbody >div > div > div:nth-child(7)',
    cvesCvssScoreCol: '.rt-table > .rt-tbody > div > .rt-tr.-odd > div:nth-child(4) > div > span',
    tableRows: '.rt-tr',
    tableBodyRows: '.rt-tbody .rt-tr',
    tableColumn: '.rt-th.leading-normal > div',
    tableBodyColumn: '.rt-tr-group:nth-child(1) > .rt-tr > .rt-td',
    tableColumnLinks: '.rt-tr-group:nth-child(1)> .rt-tr > .rt-td > a',
    allCVEColumnLink: '[data-testid="allCvesLink"]',
    fixableCVELink: '[data-testid="fixableCvesLink"]',
    numCVEColLink: '.rt-tr > .rt-td',
    cveDescription: '[data-testid="cve-description"]',
    statusChips: '[data-testid="label-chip"]',
    deploymentCountLink: '[data-test-id="deploymentCountLink"]',
    policyCountLink: '[data-test-id="policyCountLink"]',
    imageCountLink: '[data-test-id="imageCountLink"]',
    componentCountLink: '[data-test-id="componentCountLink"]'
};

export const sidePanelListEntityPageSelectors = {
    entityRowHeader:
        '[data-test-id="side-panel"] > .h-full > .flex > .flex-no-wrap > .flex > [data-test-id="panel-header"]',
    sidePanelTableBodyRows: '[data-test-id="side-panel"] .rt-tbody .rt-tr',
    parentEntityInfoHeader: '[data-test-id="breadcrumb-link-text"] > a',
    childEntityInfoHeader: '[data-test-id="breadcrumb-link-text"] > span',
    entityOverview: '[data-testid="entity-overview"]',
    metadataClusterValue: '[data-testid="Cluster-value"]',
    tileLinkText: '[data-test-id="tileLinkSuperText"]',
    tileLinkValue: '[data-test-id="tile-link-value"]',
    imageTileLink: '[data-test-id="IMAGE-tile-link"]',
    namespaceTileLink: '[data-test-id="NAMESPACE-tile-link"]',
    componentTileLink: '[data-test-id="COMPONENT-tile-link"]',
    deploymentTileLink: "[data-test-id='DEPLOYMENT-tile-link']",
    policyTileLink: "[data-test-id='POLICY-tile-link']",
    cveTileLink: '[data-test-id="CVE-tile-link"]',
    tabButton: '[data-test-id="tab"]',
    getSidePanelTabHeader: title => {
        return `[data-test-id="widget-header"] > .w-full:contains('${title}')`;
    },
    emptyFindingsSection: '[data-testid="results-message"]',
    deploymentCountText: '.rt-td [data-testid="deploymentCountText"]',
    imageCountText: '.rt-td [data-testid="imageCountText"]'
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
        return `[data-test-id="tile-link-value"]:contains('${title}')`;
    },
    widgetBody: '[data-testid="widget-body"]',
    viewAllButton: 'button:contains("View All")',
    dataRowLink: '[data-testid="numbered-list-item-name"]',
    topMostRowMCV:
        '#capture-dashboard > div > div > div > .h-full > div > div > svg > g > text:nth-child(20)',
    topMostRowFVP:
        '#capture-dashboard > div > div:nth-child(3) > div > .h-full > div > div > svg > g > text:nth-child(2)',
    entityPageHeader: '[data-test-id="header-text"]',
    topMostRowRDV:
        '#capture-dashboard > div > div:nth-child(4) > div > .h-full > div > ul > li:nth-child(1) > a > span',
    topMostRowMSPV:
        '#capture-dashboard > div > div:nth-child(6) > div > .h-full > div > ul > li:nth-child(1) > a > span',
    tabLinks: '[data-test-id="tab"]',
    allTileLinks: '#capture-widgets > div > .h-full > div > ul > li',
    tabHeader: '[data-test-id="panel-header"]'
};

const linkSelectors = {
    allCvesLink: '[data-testid="allCvesLink"]',
    fixableCvesLink: '[data-testid="fixableCvesLink"]',
    tileLinks: "[data-test-id='tile-link']",
    tileLinkValue: "[data-test-id='tile-link-value']",
    tileLinkSuperText: '[data-test-id="tileLinkSuperText"]'
};

const sidepanelSelectors = {
    backButton: '[data-testid="sidepanelBackButton"]',
    entityIcon: '[data-testid="entity-icon"]',
    sidePanelExpandButton: '[data-test-id = "external-link"]',
    getSidePanelTabLink: title => {
        return `[data-test-id="tab"]:contains('${title}')`;
    }
};

const policySidePanelSelectors = {
    policyEditButton: '[data-testid="button-link"]',
    policyEditPageHeader: '[data-test-id="side-panel-header"]'
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...linkSelectors,
    ...sidepanelSelectors,
    ...sidePanelListEntityPageSelectors,
    ...policySidePanelSelectors,
    ...tablePaginationSelectors,
    ...sidePanelSelectors,
    ...vmHomePageSelectors
};
