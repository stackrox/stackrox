import tableSelectors from '../selectors/table';
import selectSelectors from '../selectors/select';
import paginationSelectors from '../selectors/pagination';
import tooltipSelectors from '../selectors/tooltip';
import navigationSelectors from '../selectors/navigation';
import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/risk';

/*
// TODO after PatternFly conversion: update if relevant or delete if not relevant.
export const errorMessages = {
    deploymentNotFound: 'Deployment not found',
    riskNotFound: 'Risk not found',
    processNotFound: 'No processes discovered',
};
*/

const sidePanel = scopeSelectors('[data-testid="panel"]:eq(1)', {
    panelHeader: '[data-testid="panel-header"]',
    firstProcessCard: scopeSelectors('[data-testid="process-discovery-card"]:first', {
        header: '[data-testid="process"]',
        tags: {
            input: '[data-testid="process-tags"] input',
            values: '[data-testid="process-tags"] .pf-c-chip-group div.pf-c-chip',
            removeValueButton: (tag) =>
                `[data-testid="process-tags"] div.pf-c-chip:contains(${tag}) button`,
        },
    }),

    tabs: 'button[data-testid="tab"]',
    riskIndicatorsTab: 'button[data-testid="tab"]:contains("Risk Indicators")',
    deploymentDetailsTab: 'button[data-testid="tab"]:contains("Deployment Details")',
    processDiscoveryTab: 'button[data-testid="tab"]:contains("Process Discovery")',

    cancelButton: 'button[data-testid="cancel"]',
});

const eventSelectors = {
    policyViolation: '[data-testid="policy-violation-event"]',
    processActivity: '[data-testid="process-activity-event"]',
    processInBaselineActivity: '[data-testid="process-in-baseline-activity-event"]',
    restart: '[data-testid="restart-event"]',
    termination: '[data-testid="termination-event"]',
};

const clusteredEventSelectors = {
    generic: '[data-testid="clustered-generic-event"]',
    policyViolation: '[data-testid="clustered-policy-violation-event"]',
    processActivity: '[data-testid="clustered-process-activity-event"]',
    processInBaselineActivity: '[data-testid="clustered-process-in-baseline-activity-event"]',
    restart: '[data-testid="clustered-restart-event"]',
    termination: '[data-testid="clustered-termination-event"]',
};

const eventTimelineOverviewSelectors = scopeSelectors('[data-testid="event-timeline-overview"]', {
    eventCounts: '[data-testid="tile-content"] [data-testid="tileLinkSuperText"]',
    totalNumEventsText: '[data-testid="tile-content"]:first [data-testid="tile-link-value"]',
});

const eventTimelineSelectors = scopeSelectors('[data-testid="event-timeline"]', {
    panelHeader: scopeSelectors('[data-testid="event-timeline-header"]', {
        header: '[data-testid="header"]',
    }),
    backButton: '[data-testid="timeline-back-button"]',
    select: selectSelectors.singleSelect,
    legend: '[data-testid="timeline-legend"]',
    timeline: scopeSelectors('[data-testid="timeline-graph"]', {
        namesList: scopeSelectors('ul[data-testid="timeline-names-list"]', {
            listOfNames: '> li',
            firstListedName: '> li:first [data-testid="header"]',
            drillDownButtonInFirstRow: '[data-testid="timeline-drill-down-button"]:first',
        }),
        pagination: paginationSelectors,
        mainView: scopeSelectors('[data-testid="timeline-main-view"]', {
            event: eventSelectors,
            clusteredEvent: clusteredEventSelectors,
            allClusteredEvents: '[data-testid="timeline-clustered-event-marker"]',
        }),
    }),
});

export const selectors = {
    risk: `${navigationSelectors.navLinks}:contains("Risk")`,
    errMgBox: 'div.error-message',
    panel: '[data-testid="panel"]',
    panelTabs: {
        riskIndicators: 'button[data-testid="tab"]:contains("Risk Indicators")',
        deploymentDetails: 'button[data-testid="tab"]:contains("Deployment Details")',
        processDiscovery: 'button[data-testid="tab"]:contains("Process Discovery")',
    },
    cancelButton: 'button[data-testid="cancel"]',
    search: {
        valueContainer: '.react-select__value-container',
        searchLabels: '.react-select__multi-value__label',
        // selectors for legacy tests
        searchModifier: '.react-select__multi-value__label:first',
        searchWord: '.react-select__multi-value__label:eq(1)',
    },
    createPolicyButton:
        '[data-testid="panel-button-create-policy-from-search"]:contains("Create Policy")',
    mounts: {
        label: 'div:contains("Mounts"):last',
        items: 'div:contains("Mounts"):last + ul li div',
    },
    imageLink: 'div:contains("Image Name") + a',
    table: scopeSelectors('[data-testid="panel"]:first', tableSelectors),
    collapsible: {
        card: '.Collapsible',
        header: '.Collapsible__trigger',
        body: '.Collapsible__contentInner',
    },
    suspiciousProcesses: "[data-testid='suspicious-process']",
    viewDeploymentsInNetworkGraphButton: '[data-testid="view-deployments-in-network-graph-button"]',
    sidePanel,
    eventTimeline: eventTimelineSelectors,
    eventTimelineOverview: eventTimelineOverviewSelectors,
    eventTimelineOverviewButton: 'button[data-testid="event-timeline-overview"]',
    tooltip: {
        ...tooltipSelectors,
        legendContents: `${tooltipSelectors.overlay} > div`,
        legendContent: {
            event: eventSelectors,
        },
        bodyContent: scopeSelectors(tooltipSelectors.body, {
            uidFieldValue: `[data-testid="tooltip-uid-field-value"]`,
            eventDetails: 'ul > li',
        }),
    },
};
