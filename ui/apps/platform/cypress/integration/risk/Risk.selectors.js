import tableSelectors from '../../selectors/table';
import selectSelectors from '../../selectors/select';
import tooltipSelectors from '../../selectors/tooltip';
import navigationSelectors from '../../selectors/navigation';
import scopeSelectors from '../../helpers/scopeSelectors';

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
        mainView: scopeSelectors('[data-testid="timeline-main-view"]', {
            event: eventSelectors,
            clusteredEvent: clusteredEventSelectors,
            allClusteredEvents: '[data-testid="timeline-clustered-event-marker"]',
        }),
    }),
});

export const selectors = {
    risk: `${navigationSelectors.navLinks}:contains("Risk")`,
    panel: '[data-testid="panel"]',
    search: {
        valueContainer: '.react-select__value-container',
        searchLabels: '.react-select__multi-value__label',
        // selectors for legacy tests
        searchModifier: '.react-select__multi-value__label:first',
        searchWord: '.react-select__multi-value__label:eq(1)',
    },
    createPolicyButton: 'button:contains("Create policy")',
    imageLink: 'div:contains("Image Name") + a',
    table: scopeSelectors('[data-testid="panel"]:first', tableSelectors),
    eventTimeline: eventTimelineSelectors,
    tooltip: {
        ...tooltipSelectors,
        legendContents: `${tooltipSelectors.overlay} .pf-c-tooltip__content`,
        legendContent: {
            event: eventSelectors,
        },
        getUidFieldIconSelector: (type) =>
            `.pf-c-tooltip__content span:contains("UID") ~ svg[fill="var(--pf-global--${type}-color--100)"]`,
        bodyContent: scopeSelectors(tooltipSelectors.body, {
            eventDetails: 'ul > li',
        }),
    },
};
