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

    cy.get('h1:contains("Administration Events")');
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

// interact

export function interactAndWaitForAdministrationEvents(interactionCallback, staticResponseMap) {
    interactAndWaitForResponses(
        interactionCallback,
        routeMatcherMapForAdministationEvents,
        staticResponseMap
    );

    cy.get('h1:contains("Administration Events")');
}

// query

export function getFilterQueryForPage(key, value) {
    return `s[${encodeURI(key)}]=${encodeURI(value)}`;
}

// selector

export function getDescriptionListGroupSelector(term, description) {
    return `dl:has('dt:contains("${term}")') dd:contains("${description}")`;
}

export function getDescriptionListTermSelector(term) {
    return `dl:has('dt:contains("${term}")')`;
}

function getToggleSelector(label) {
    return `button[aria-label="${label}"]`;
}

export function selectFilter(label, item) {
    const toggleSelector = getToggleSelector(label);
    cy.get(toggleSelector).click();
    cy.get(`button.pf-v6-c-menu__item:contains("${item}")`).click();
}
