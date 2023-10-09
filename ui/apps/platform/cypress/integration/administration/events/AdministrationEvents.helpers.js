import { interactAndWaitForResponses } from '../../../helpers/request';
import { visit } from '../../../helpers/visit';

// Source of truth for keys in routeMatcherMap and staticResponseMap objects.
export const eventAlias = 'administration/events/id';
export const eventsAlias = 'administration/events';
export const eventsCountAlias = 'count/administration/events';

const routeMatcherMapForAdministationEvents = {
    [eventsAlias]: {
        method: 'GET',
        url: '/v1/administration/events?*',
    },
    [eventsCountAlias]: {
        method: 'GET',
        url: '/v1/count/administration/events?*',
    },
};

const routeMatcherMapForAdministationEvent = {
    [eventAlias]: {
        method: 'GET',
        url: '/v1/administration/events/*',
    },
};

const basePath = '/main/administration-events';

// visit

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAdministrationEvents(staticResponseMap) {
    visit(basePath, routeMatcherMapForAdministationEvents, staticResponseMap);

    cy.get(`h1:contains("Administration Events")`);
}

/**
 * @param {Record<string, { body: unknown } | { fixture: string }>} [staticResponseMap]
 */
export function visitAdministrationEventFromTableRow(index0, staticResponseMap) {
    interactAndWaitForResponses(
        () => {
            cy.get(`tbody tr:nth-child(${index0 + 1}) td[data-label="Domain"] a`).click();
        },
        routeMatcherMapForAdministationEvent,
        staticResponseMap
    );
}

// assert

export function assertDescriptionListGroup(term, description) {
    cy.get(`dl:has('dt:contains("${term}")') dd:contains("${description}")`);
}
