import { selectors as tablePaginationSelectors } from './TablePagination';
import panelSelectors from '../selectors/panel';
import tableSelectors from '../selectors/table';
import scopeSelectors from '../helpers/scopeSelectors';
import navigationSelectors from '../selectors/navigation';

const baseURL = '/main/vulnerability-management';

export const url = {
    dashboard: baseURL,
    list: {
        policies: `${baseURL}/policies`,
        clusters: `${baseURL}/clusters`,
        namespaces: `${baseURL}/namespaces`,
        deployments: `${baseURL}/deployments`,
        images: `${baseURL}/images`,
        nodes: `${baseURL}/nodes`,
        components: `${baseURL}/components`,
        'image-components': `${baseURL}/image-components`,
        'node-components': `${baseURL}/node-components`,
        cves: `${baseURL}/cves`,
        'image-cves': `${baseURL}/image-cves`,
        'node-cves': `${baseURL}/node-cves`,
        'cluster-cves': `${baseURL}/cluster-cves`,
        image: `${baseURL}/image`,
        cve: `${baseURL}/cve`,
        policy: `${baseURL}/policy`,
        deployment: `${baseURL}/deployment`,
    },
    sidepanel: {
        image: `${baseURL}/images?workflowState[0][t]=IMAGE&workflowState[0][i]=sha256:02382353821b12c21b062c59184e227e001079bb13ebd01f9d3270ba0fcbf1e4`,
        node: `${baseURL}/nodes?workflowState[0][t]=NODE&workflowState[0][i]=065fe8cb-d9af-4516-a91e-3941e9db58ca`,
    },
    reporting: {
        list: `${baseURL}/reports`,
        create: `${baseURL}/reports?action=create`,
    },
};

/*
 * Headings on entities pages: uppercase style hides the inconsistencies.
 * The keys correspond to url list object above.
 */
export const headingPlural = {
    clusters: 'clusters',
    components: 'components',
    'image-components': 'image components',
    'node-components': 'node components',
    cves: 'CVES',
    'image-cves': 'Image CVES',
    'node-cves': 'Node CVES',
    'cluster-cves': 'Platform CVES',
    deployments: 'deployments',
    images: 'images',
    namespaces: 'namespaces',
    nodes: 'nodes',
    policies: 'policies',
};

export const vmHomePageSelectors = {
    // TODO: remove this selector, after at least one sub-menu is added to Vuln Mgmt menu
    vulnManagementNavLink: `${navigationSelectors.navLinks}:contains("Vulnerability Management")`,

    // the selectors below are for when the Vulm Mgmt menu item is expandable
    vulnManagementExpandableNavLink: `${navigationSelectors.navExpandable}:contains("Vulnerability Management")`,
    vulnManagementExpandedDashboardNavLink: `${navigationSelectors.nestedNavLinks}:contains("Dashboard")`,
    vulnManagementExpandedReportingNavLink: `${navigationSelectors.nestedNavLinks}:contains("Reporting")`,
};
export const listSelectors = {
    riskScoreCol: '.rt-table > .rt-tbody > div > div > div:nth-child(10)',
    componentsRiskScoreCol: '.rt-table > .rt-tbody >div > div > div:nth-child(7)',
    cvesCvssScoreCol: '.rt-table > .rt-tbody > div > .rt-tr.-odd > div:nth-child(4) > div > span',
    tableRows: '.rt-tr',
    tableCells: '.rt-td',
    tableBodyRowGroups: '.rt-tbody .rt-tr-group',
    tableBodyRows: '.rt-tbody .rt-tr',
    tableRowCheckbox: '[data-testid="checkbox-table-row-selector"]',
    tableColumn: '.rt-th.leading-normal > div',
    tableBodyColumn: '.rt-tr-group:nth-child(1) > .rt-tr > .rt-td',
    tableColumnLinks: '.rt-tr-group:nth-child(1) > .rt-tr > .rt-td a',
    allCVEColumnLink: '[data-testid="allCvesLink"]',
    fixableCVELink: '[data-testid="fixableCvesLink"]',
    numCVEColLink: '.rt-tr > .rt-td',
    cveDescription: '[data-testid="cve-description"]',
    statusChips: '[data-testid="label-chip"]',
    deploymentCountLink: '[data-testid="deploymentCountLink"]',
    failingDeploymentCountLink: '[data-testid="failingDeploymentsCountLink"]',
    policyCountLink: '[data-testid="policyCountLink"]',
    imageCountLink: '[data-testid="imageCountLink"]',
    componentCountLink: '[data-testid="componentCountLink"]',
    cveSuppressPanelButton: '[data-testid="panel-button-suppress-selected-cves"]',
    cveUnsuppressPanelButton: '[data-testid="panel-button-unsuppress-selected-cves"]',
    cveAddToPolicyButton: '[data-testid="panel-button-add-cves-to-policy"]',
    cveAddToPolicyShortForm: {
        // TODO: fix the following selector for react-select, that evil component
        select: '[data-testid="policy-short-form"] select',
        selectValue: '[data-testid="policy-short-form"] .react-select__multi-value__label',
    },
    suppressOneDayOption: '[data-testid="1 Day"]',
    suppressToggleViewPanelButton: '[data-testid="panel-button-toggle-suppressed-cves-view"]',
    cveUnsuppressRowButton: '[data-testid="row-action-unsuppress"]',
    cveTypes: '.rt-tbody [data-testid="cve-type"]',
};

export const sidePanelListEntityPageSelectors = {
    entityRowHeader: '[data-testid="side-panel"] [data-testid="panel-header"]',
    sidePanelTableBodyRows: '[data-testid="side-panel"] .rt-tbody .rt-tr',
    parentEntityInfoHeader: '[data-testid="breadcrumb-link-text"] > a',
    childEntityInfoHeader: '[data-testid="breadcrumb-link-text"] > span',
    entityOverview: '[data-testid="entity-overview"]',
    metadataClusterValue: '[data-testid="Cluster-value"]',
    tileLinkText: '[data-testid="tileLinkSuperText"]',
    tileLinkValue: '[data-testid="tile-link-value"]',
    imageTileLink: '[data-testid="IMAGE-tile-link"]',
    namespaceTileLink: '[data-testid="NAMESPACE-tile-link"]',
    imageComponentTileLink: '[data-testid="IMAGE_COMPONENT-tile-link"]',
    nodeComponentTileLink: '[data-testid="NODE_COMPONENT-tile-link"]',
    componentTileLink: '[data-testid="COMPONENT-tile-link"]',
    deploymentTileLink: "[data-testid='DEPLOYMENT-tile-link']",
    policyTileLink: "[data-testid='POLICY-tile-link']",
    cveTileLink: '[data-testid="CVE-tile-link"]',
    nodeTileLink: '[data-testid="NODE-tile-link"]',
    tabButton: '[data-testid="tabs"] button',
    getSidePanelTabHeader: (title) => {
        return `[data-testid="widget-header"] > .w-full:contains('${title}')`;
    },
    emptyFindingsSection: '[data-testid="results-message"]',
    deploymentCountText: '.rt-td [data-testid="deploymentCountText"]',
    imageCountText: '.rt-td [data-testid="imageCountText"]',
    cveType: '[data-testid="entity-overview"] [data-testid="cve-type"]',
};

export const dashboardSelectors = {
    applicationAndInfrastructureDropdown: 'button:contains("Application & Infrastructure")',
    topRiskyItems: {
        select: {
            input: '[data-testid="widget"] .react-select__control',
            value: '[data-testid="widget"] .react-select__single-value',
            options: '[data-testid="widget"] .react-select__option',
        },
    },
    getMenuListItem: (name) => {
        return `[data-testid="menu-list"] [data-testid="${name}"]`;
    },
    getWidget: (title) => {
        return `[data-testid="widget"]:contains('${title}')`;
    },
    getTileLink: (title) => {
        return `[data-testid="tile-link"]:contains('${title}')`;
    },
    getAllClickableTileLinks: (title) => {
        return `[data-testid="tile-link-value"]:contains('${title}')`;
    },
    widgetBody: '[data-testid="widget-body"]',
    viewAllButton: 'a:contains("View All")',
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
    tabHeader: '[data-testid="panel-header"]',
};

const linkSelectors = {
    allCvesLink: '[data-testid="allCvesLink"]',
    fixableCvesLink: '[data-testid="fixableCvesLink"]',
    tileLinks: "[data-testid='tile-link']",
    tileLinkValue: "[data-testid='tile-link-value']",
    tileLinkSuperText: '[data-testid="tileLinkSuperText"]',
};

const sidePanelSelectors = {
    backButton: '[data-testid="sidepanelBackButton"]',
    entityIcon: '[data-testid="entity-icon"]',
    sidePanelExpandButton: '[data-testid = "external-link"]',
    getSidePanelTabLink: (title) => {
        return `[data-testid="tab"]:contains('${title}')`;
    },
    policyFindingsSection: scopeSelectors('[data-testid="policy-findings-section"]', {
        table: tableSelectors,
    }),
    violationTags: {
        input: '[data-testid="violation-tags"] input',
        values: '[data-testid="violation-tags"] .pf-c-chip-group div.pf-c-chip',
        removeValueButton: (tag) =>
            `[data-testid="violation-tags"] div.pf-c-chip:contains(${tag}) button`,
    },
    scanDataMessage: '[data-testid="message"].error-message:contains("CVE Data May Be Inaccurate")',
};

const policySidePanelSelectors = {
    policyEditButton: '[data-testid="button-link"]',
    policyEditPageHeader: '[data-testid="side-panel-header"]',
};

const reportSection = {
    pageTitle: 'h1',
    createReportLink: 'a:contains("Create report")',
    breadcrumbItems: '.pf-c-breadcrumb__item',
    buttons: {
        create: 'button:contains("Create")',
        cancel: 'button:contains("Cancel")',
    },
    table: {
        column: {
            name: 'th:contains("Report")',
            description: 'th:contains("Description")',
            cveFixabilityType: 'th:contains("CVE fixability type")',
            cveSeverities: 'th:contains("CVE severities")',
            lastRun: 'th:contains("Last run")',
        },
        rows: 'tbody tr',
    },
};

export const selectors = {
    ...dashboardSelectors,
    ...listSelectors,
    ...linkSelectors,
    ...sidePanelSelectors,
    ...sidePanelListEntityPageSelectors,
    ...policySidePanelSelectors,
    ...tablePaginationSelectors,
    ...panelSelectors,
    ...vmHomePageSelectors,
    // TODO-ivan: unscrew everything above, it overrides each other etc., move to scoped definitions
    mainTable: scopeSelectors('[data-testid="panel"]', tableSelectors),
    sidePanel1: scopeSelectors(panelSelectors.sidePanel, sidePanelSelectors),
    reportSection,
};
