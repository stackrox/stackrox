import { format } from 'date-fns';

import { graphql } from '../../constants/apiEndpoints';
import { interactAndVisitNetworkGraphWithDeploymentSelected } from '../../helpers/networkGraph';
import { interactAndWaitForResponses } from '../../helpers/request';
import { visit } from '../../helpers/visit';

// visit

const riskURL = '/main/risk';

export const deploymentswithprocessinfoAlias = 'deploymentswithprocessinfo';
export const deploymentscountAlias = 'deploymentscount';
export const searchOptionsAlias = 'searchOptions';

const routeMatcherMap = {
    [deploymentswithprocessinfoAlias]: {
        method: 'GET',
        url: '/v1/deploymentswithprocessinfo*', // wildcard for ?query=
    },
    [deploymentscountAlias]: {
        method: 'GET',
        url: '/v1/deploymentscount*',
    },
    [searchOptionsAlias]: {
        method: 'POST',
        url: graphql('searchOptions'),
    },
};

export function visitRiskDeployments() {
    visit(riskURL, routeMatcherMap);

    cy.get('h1:contains("Risk")');
}

export function visitRiskDeploymentsWithSearchQuery(search) {
    visit(`${riskURL}${search}`, routeMatcherMap);

    cy.get('h1:contains("Risk")');
}

export function viewRiskDeploymentByName(deploymentName) {
    // Assume location is risk deployments table.
    const routeMatcherMapForDeployment = {
        'deploymentswithrisk/id': {
            method: 'GET',
            url: '/v1/deploymentswithrisk/*',
        },
    };

    interactAndWaitForResponses(() => {
        // Specify nth-child(4) for Namespace to make sure it is stackrox.
        // For example, prevent multiple matches with collector in gmp-system namespace.
        //
        // Specify nth-child(1) for Name of deployment.
        //
        // Call contains method with RegExp for exact match and only first element.
        // Unlike contains pseudo-selector which would require :nth(0) in case of multiple matches.
        cy.get(
            `.rt-tbody .rt-tr:has('.rt-td:nth-child(4):contains("stackrox")') .rt-td:nth-child(1)`
        )
            .contains(new RegExp(`^${deploymentName}$`))
            .click();
    }, routeMatcherMapForDeployment);

    // Unlike some classic containers, risk list header has different data-testid attribute.
    cy.get(`[data-testid="panel-header"]:contains("${deploymentName}")`);
}

export function viewRiskDeploymentInNetworkGraph() {
    interactAndVisitNetworkGraphWithDeploymentSelected(() => {
        cy.get('a:contains("View Deployment in Network Graph")').click();
    });
}

// Process Discovery

const getDeploymentEventTimelineAlias = 'getDeploymentEventTimeline';

const routeMatcherMapForDeploymentEventTimeline = {
    [getDeploymentEventTimelineAlias]: {
        method: 'POST',
        url: graphql('getDeploymentEventTimeline'),
    },
};

export function viewGraph(fixtureForDeploymentEventTimeline) {
    interactAndWaitForResponses(
        () => {
            cy.get('button:contains("View Graph")').click();
        },
        routeMatcherMapForDeploymentEventTimeline,
        fixtureForDeploymentEventTimeline && {
            [getDeploymentEventTimelineAlias]: {
                fixture: fixtureForDeploymentEventTimeline,
            },
        }
    );

    cy.get('[data-testid="event-timeline"]');
}

const nextPageSelector = '[aria-label="Modal"] button[aria-label="Go to next page"]';

export function clickNextPageInEventTimelineWithRequest(fixtureForDeploymentEventTimeline) {
    interactAndWaitForResponses(
        () => {
            // Risk deployments list also has a button.
            cy.get(nextPageSelector).click();
        },
        routeMatcherMapForDeploymentEventTimeline,
        fixtureForDeploymentEventTimeline && {
            [getDeploymentEventTimelineAlias]: {
                fixture: fixtureForDeploymentEventTimeline,
            },
        }
    );
}

export function clickNextPageInEventTimelineWithoutRequest() {
    cy.get(nextPageSelector).click();
}

/**
 * Finds an event based on the event id and returns the formatted timestamp
 * @param {string} id - the event id
 * @returns {Promise<string>} - a promise that, once resolved, will return the formatted timestamp of an event for the specified event typee
 */
export function getFormattedEventTimeById(id, fixtureForDeploymentEventTimeline) {
    return cy.fixture(fixtureForDeploymentEventTimeline).then((json) => {
        const eventTime = json.data.pods[0].events.find((event) => event.id === id).timestamp;
        return `Event time: ${format(eventTime, 'MM/DD/YYYY | h:mm:ssA')}`;
    });
}

const getPodEventTimelineAlias = 'getPodEventTimeline';

const routeMatcherMapForPodEventTimeline = {
    [getPodEventTimelineAlias]: {
        method: 'POST',
        url: graphql('getPodEventTimeline'),
    },
};

export function clickFirstDrillDownButtonInEventTimeline(fixtureForPodEventTimeline) {
    interactAndWaitForResponses(
        () => {
            cy.get('[data-testid="timeline-drill-down-button"]:nth(0)').click();
        },
        routeMatcherMapForPodEventTimeline,
        fixtureForPodEventTimeline && {
            [getPodEventTimelineAlias]: {
                fixture: fixtureForPodEventTimeline,
            },
        }
    );
}

// interact

export function clickTab(tabText) {
    cy.get(`button[data-testid="tab"]:contains("${tabText}")`).click();
}

export function filterEventsByType(eventType) {
    cy.get('[aria-label="Modal"] .react-select__control').click();
    cy.get(`[aria-label="Modal"] .react-select__option:contains("${eventType}")`).click();
}
