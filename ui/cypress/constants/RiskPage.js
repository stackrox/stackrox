import tableSelectors from '../selectors/table';
import { processTagsSelectors } from '../selectors/tags';
import { processCommentsSelectors, commentsDialogSelectors } from '../selectors/comments';
import selectSelectors from '../selectors/select';
import paginationSelectors from '../selectors/pagination';
import tooltipSelectors from '../selectors/tooltip';
import scopeSelectors from '../helpers/scopeSelectors';

export const url = '/main/risk';

export const errorMessages = {
    deploymentNotFound: 'Deployment not found',
    riskNotFound: 'Risk not found',
    processNotFound: 'No processes discovered',
};

const sidePanelSelectors = scopeSelectors('[data-testid="panel"]:eq(1)', {
    firstProcessCard: scopeSelectors('[data-testid="process-discovery-card"]:first', {
        header: '[data-testid="process"]',
        tags: processTagsSelectors,
        comments: processCommentsSelectors,
    }),

    riskIndicatorsTab: 'button[data-testid="tab"]:contains("Risk Indicators")',
    deploymentDetailsTab: 'button[data-testid="tab"]:contains("Deployment Details")',
    processDiscoveryTab: 'button[data-testid="tab"]:contains("Process Discovery")',

    cancelButton: 'button[data-testid="cancel"]',
});

const eventSelectors = {
    policyViolation: ' [data-testid="policy-violation-event"]',
    processActivity: '[data-testid="process-activity-event"]',
    whitelistedProcessActivity: '[data-testid="whitelisted-process-activity-event"]',
    restart: '[data-testid="restart-event"]',
    termination: '[data-testid="termination-event"]',
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
            eventsInFirstRow:
                '[data-testid="timeline-events-row"]:first [data-testid="timeline-event-marker"]',
            allEvents: '[data-testid="timeline-event-marker"]',
        }),
    }),
});

export const selectors = {
    risk: 'nav.left-navigation li:contains("Risk") a',
    errMgBox: 'div.error-message',
    panelTabs: {
        riskIndicators: 'button[data-testid="tab"]:contains("Risk Indicators")',
        deploymentDetails: 'button[data-testid="tab"]:contains("Deployment Details")',
        processDiscovery: 'button[data-testid="tab"]:contains("Process Discovery")',
    },
    cancelButton: 'button[data-testid="cancel"]',
    search: {
        searchLabels: '.react-select__multi-value__label',
        // selectors for legacy tests
        searchModifier: '.react-select__multi-value__label:first',
        searchWord: '.react-select__multi-value__label:eq(1)',
    },
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
    networkNodeLink: '[data-testid="network-node-link"]',
    sidePanel: sidePanelSelectors,
    commentsDialog: commentsDialogSelectors,
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
        }),
    },
};
