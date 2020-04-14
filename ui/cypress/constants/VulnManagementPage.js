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
    vmDBPageTileLink: '[data-testid="Vulnerability Management"]'
};
export const listSelectors = {
    riskScoreCol: '.rt-table > .rt-tbody > div > div > div:nth-child(10)',
    componentsRiskScoreCol: '.rt-table > .rt-tbody >div > div > div:nth-child(7)',
    cvesCvssScoreCol: '.rt-table > .rt-tbody > div > .rt-tr.-odd > div:nth-child(4) > div > span',
    tableRows: '.rt-tr',
    tableBodyRows: '.rt-tbody .rt-tr',
    tableRowCheckbox: '[data-testid="checkbox-table-row-selector"]',
    tableColumn: '.rt-th.leading-normal > div',
    tableBodyColumn: '.rt-tr-group:nth-child(1) > .rt-tr > .rt-td',
    tableColumnLinks: '.rt-tr-group:nth-child(1)> .rt-tr > .rt-td > a',
    allCVEColumnLink: '[data-testid="allCvesLink"]',
    fixableCVELink: '[data-testid="fixableCvesLink"]',
    numCVEColLink: '.rt-tr > .rt-td',
    cveDescription: '[data-testid="cve-description"]',
    statusChips: '[data-testid="label-chip"]',
    deploymentCountLink: '[data-testid="deploymentCountLink"]',
    policyCountLink: '[data-testid="policyCountLink"]',
    imageCountLink: '[data-testid="imageCountLink"]',
    componentCountLink: '[data-testid="componentCountLink"]',
    cveSuppressPanelButton: '[data-testid="panel-button-suppress-selected-cves"]',
    cveUnsuppressPanelButton: '[data-testid="panel-button-unsuppress-selected-cves"]',
    suppressOneHourOption: '[data-testid="1 Hour"]',
    suppressToggleViewPanelButton: '[data-testid="panel-button-toggle-suppressed-cves-view"]',
    cveUnsuppressRowButton: '[data-testid="row-action-unsuppress"]'
};

export const sidePanelListEntityPageSelectors = {
    entityRowHeader:
        '[data-testid="side-panel"] > .h-full > .flex > .flex-no-wrap > .flex > [data-testid="panel-header"]',
    sidePanelTableBodyRows: '[data-testid="side-panel"] .rt-tbody .rt-tr',
    parentEntityInfoHeader: '[data-testid="breadcrumb-link-text"] > a',
    childEntityInfoHeader: '[data-testid="breadcrumb-link-text"] > span',
    entityOverview: '[data-testid="entity-overview"]',
    metadataClusterValue: '[data-testid="Cluster-value"]',
    tileLinkText: '[data-testid="tileLinkSuperText"]',
    tileLinkValue: '[data-testid="tile-link-value"]',
    imageTileLink: '[data-testid="IMAGE-tile-link"]',
    namespaceTileLink: '[data-testid="NAMESPACE-tile-link"]',
    componentTileLink: '[data-testid="COMPONENT-tile-link"]',
    deploymentTileLink: "[data-testid='DEPLOYMENT-tile-link']",
    policyTileLink: "[data-testid='POLICY-tile-link']",
    cveTileLink: '[data-testid="CVE-tile-link"]',
    tabButton: '[data-testid="tab"]',
    getSidePanelTabHeader: title => {
        return `[data-testid="widget-header"] > .w-full:contains('${title}')`;
    },
    emptyFindingsSection: '[data-testid="results-message"]',
    deploymentCountText: '.rt-td [data-testid="deploymentCountText"]',
    imageCountText: '.rt-td [data-testid="imageCountText"]'
};

export const dashboardSelectors = {
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    topRiskyItems: {
        select: {
            input: '[data-testid="widget"] .react-select__control',
            value: '[data-testid="widget"] .react-select__single-value',
            options: '[data-testid="widget"] .react-select__option'
        }
    },
    getMenuListItem: name => {
        return `[data-testid="menu-list"] [data-testid="${name}"]`;
    },
    getWidget: title => {
        return `[data-testid="widget"]:contains('${title}')`;
    },
    getTileLink: title => {
        return `[data-testid="tile-link"]:contains('${title}')`;
    },
    getAllClickableTileLinks: title => {
        return `[data-testid="tile-link-value"]:contains('${title}')`;
    },
    widgetBody: '[data-testid="widget-body"]',
    viewAllButton: 'button:contains("View All")',
    dataRowLink: '[data-testid="numbered-list-item-name"]',
    topMostRowMCV:
        '#capture-dashboard > div > div > div > .h-full > div > div > svg > g > text:nth-child(20)',
    topMostRowFVP:
        '#capture-dashboard > div > div:nth-child(3) > div > .h-full > div > div > svg > g > text:nth-child(2)',
    entityPageHeader: '[data-testid="header-text"]',
    topMostRowRDV:
        '#capture-dashboard > div > div:nth-child(4) > div > .h-full > div > ul > li:nth-child(1) > a > span',
    topMostRowMSPV:
        '#capture-dashboard > div > div:nth-child(6) > div > .h-full > div > ul > li:nth-child(1) > a > span',
    tabLinks: '[data-testid="tab"]',
    allTileLinks: '#capture-widgets > div > .h-full > div > ul > li',
    tabHeader: '[data-testid="panel-header"]'
};

const linkSelectors = {
    allCvesLink: '[data-testid="allCvesLink"]',
    fixableCvesLink: '[data-testid="fixableCvesLink"]',
    tileLinks: "[data-testid='tile-link']",
    tileLinkValue: "[data-testid='tile-link-value']",
    tileLinkSuperText: '[data-testid="tileLinkSuperText"]'
};

const sidepanelSelectors = {
    backButton: '[data-testid="sidepanelBackButton"]',
    entityIcon: '[data-testid="entity-icon"]',
    sidePanelExpandButton: '[data-testid = "external-link"]',
    getSidePanelTabLink: title => {
        return `[data-testid="tab"]:contains('${title}')`;
    }
};

const policySidePanelSelectors = {
    policyEditButton: '[data-testid="button-link"]',
    policyEditPageHeader: '[data-testid="side-panel-header"]'
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
