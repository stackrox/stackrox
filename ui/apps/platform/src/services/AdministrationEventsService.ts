import qs from 'qs';

import { ApiSortOption, SearchFilter } from 'types/search';
import { SortOption } from 'types/table';

import axios from './instance';

import { Pagination } from './types';

const eventsCountUrl = '/v1/count/administration/events';
const eventsUrl = '/v1/administration/events';

/*
 * Especially to prevent confusion which id and type value,
 * we recommend destructuring assignment to local variable names:
 *
 * const { id: resourceID, name: resourceName, type: resourceType } = resource;
 */
export type AdministrationEventResource = {
    type: string;
    id: string;
    name: string;
};

export type AdministrationEvent = {
    id: string;
    type: AdministrationEventType;
    level: AdministrationEventLevel;
    message: string;
    hint: string;
    domain: string;
    resource: AdministrationEventResource;
    numOccurrences: string; // int64
    lastOccurredAt: string; // ISO 8601
    createdAt: string; // ISO 8601
};

// types

// Order of items is for search filter options.
const types = [
    'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE',
    'ADMINISTRATION_EVENT_TYPE_GENERIC',
    'ADMINISTRATION_EVENT_TYPE_UNKNOWN',
] as const; // for isType function

export type AdministrationEventType = (typeof types)[number];

// level

// Order of items is for search filter options.
export const levels = [
    'ADMINISTRATION_EVENT_LEVEL_ERROR',
    'ADMINISTRATION_EVENT_LEVEL_WARNING',
    'ADMINISTRATION_EVENT_LEVEL_SUCCESS',
    'ADMINISTRATION_EVENT_LEVEL_INFO',
    'ADMINISTRATION_EVENT_LEVEL_UNKNOWN',
] as const; // for isLevel function

export type AdministrationEventLevel = (typeof levels)[number];

// domain

/*
 * Backend source of truth for domains:
 * https://github.com/stackrox/stackrox/blob/master/pkg/administration/events/domain.go
 */
export const domains = ['Authentication', 'General', 'Image Scanning', 'Integrations'] as const;

// resourceType

/*
 * Backend source of truth for resource types:
 * https://github.com/stackrox/stackrox/blob/master/pkg/administration/events/resources/resources.go
 * https://github.com/stackrox/stackrox/blob/master/pkg/sac/resources/list.go
 */
export const resourceTypes = ['API Token', 'Cluster', 'Image', 'Node', 'Notifier'] as const;

// filter

export type AdministrationEventsFilter = {
    from?: string; // ISO 8601 lower (older) boundary
    until?: string; // ISO 8601 upper (newer) boundary
    domain?: string[];
    resourceType?: string[];
    type?: AdministrationEventType[];
    level?: AdministrationEventLevel[];
};

// For consistency with useURLSort hook, especially in case the table columns become sortable,
// useURLSearch hook also uses search strings.
// See proto/storage/administration_event.proto
const domainField = 'Event Domain';
const levelField = 'Event Level';
const resourceTypeField = 'Resource Type';
const typeField = 'Event Type';

export type CountAdministrationEventsRequest = {
    filter: AdministrationEventsFilter;
};

// The total number of notifications after filtering and deduplication.
export type CountAdministrationEventsResponse = {
    count: number; // int32
};

export type GetAdministrationEventResponse = {
    event: AdministrationEvent;
};

export type ListAdministrationEventsRequest = {
    pagination: Pagination;
    filter: AdministrationEventsFilter;
};

export type ListAdministrationEventsResponse = {
    events: AdministrationEvent[];
};

export function countAdministrationEvents(filter: AdministrationEventsFilter): Promise<number> {
    const params = qs.stringify({ filter }, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<CountAdministrationEventsResponse>(`${eventsCountUrl}?${params}`)
        .then((response) => response.data.count);
}

export function getAdministrationEvent(id: string): Promise<AdministrationEvent> {
    return axios
        .get<GetAdministrationEventResponse>(`${eventsUrl}/${id}`)
        .then((response) => response.data.event);
}

export function listAdministrationEvents(
    arg: ListAdministrationEventsRequest
): Promise<AdministrationEvent[]> {
    const params = qs.stringify(arg, { arrayFormat: 'repeat', allowDots: true });
    return axios
        .get<ListAdministrationEventsResponse>(`${eventsUrl}?${params}`)
        .then((response) => response.data.events);
}

type GetAdministrationEventResponseArg = {
    page: number;
    perPage: number;
    searchFilter: SearchFilter;
    sortOption: ApiSortOption;
};

export function getListAdministrationEventsArg({
    page,
    perPage,
    searchFilter,
    sortOption,
}: GetAdministrationEventResponseArg): ListAdministrationEventsRequest {
    const filter = getAdministrationEventsFilter(searchFilter);
    const pagination: Pagination = { limit: perPage, offset: page - 1, sortOption };
    return { filter, pagination };
}

// Given searchFilter from useURLSearch hook, return validated filter argument.

export function getAdministrationEventsFilter(
    searchFilter: SearchFilter
): AdministrationEventsFilter {
    return {
        from: getDateTime(searchFilter.from),
        until: getDateTime(searchFilter.until),
        domain: getValue(searchFilter[domainField]),
        level: getLevel(searchFilter[levelField]),
        resourceType: getValue(searchFilter[resourceTypeField]),
        type: getType(searchFilter[typeField]),
    };
}

function hasItems(arg: unknown[] | undefined) {
    return Array.isArray(arg) && arg.length !== 0;
}

export function hasAdministrationEventsFilter(searchFilter: SearchFilter) {
    const filter = getAdministrationEventsFilter(searchFilter);
    const { from, until, domain, level, resourceType, type } = filter;
    return (
        Boolean(from) ||
        Boolean(until) ||
        hasItems(domain) ||
        hasItems(level) ||
        hasItems(resourceType) ||
        hasItems(type)
    );
}

type SearchFilterValue = string | string[] | undefined;

// from and until

function getDateTime(arg: SearchFilterValue): string | undefined {
    if (typeof arg === 'string' && isDateTime(arg)) {
        return arg;
    }

    return undefined;
}

// 20yy-mm-ddThh:mm:ssZ (excludes some but not all invalid month-day combinations).
const isDateTimeRegExp =
    /^20\d\d-(?:0\d|1[012])-(?:0[123456789]|1\d|2\d|3[01])T(?:0\d|1\d|2[0123]):[012345]\d:\d\dZ$/;

function isDateTime(arg: string) {
    return isDateTimeRegExp.test(arg);
}

// level

function getLevel(arg: SearchFilterValue): AdministrationEventLevel[] | undefined {
    if (typeof arg === 'string' && isLevel(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter(isLevel);
    }

    return undefined;
}

function isLevel(arg: string): arg is AdministrationEventLevel {
    return levels.some((level) => level === arg);
}

export function replaceSearchFilterLevel(
    searchFilter: SearchFilter,
    level: AdministrationEventLevel | undefined
): SearchFilter {
    return { ...searchFilter, [levelField]: level };
}

// type

function getType(arg: SearchFilterValue): AdministrationEventType[] | undefined {
    if (typeof arg === 'string' && isType(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter(isType);
    }

    return undefined;
}

function isType(arg: string): arg is AdministrationEventType {
    return types.includes(arg as AdministrationEventType);
}

// domain

export function replaceSearchFilterDomain(
    searchFilter: SearchFilter,
    domain: string | undefined
): SearchFilter {
    return { ...searchFilter, [domainField]: domain };
}

// resourceType

export function replaceSearchFilterResourceType(
    searchFilter: SearchFilter,
    resourceType: string | undefined
): SearchFilter {
    return { ...searchFilter, [resourceTypeField]: resourceType };
}

// domain and resourceType

function getValue(arg: SearchFilterValue): string[] | undefined {
    if (typeof arg === 'string') {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg;
    }

    return undefined;
}

// For useURLSort hook.

// See proto/storage/administration_event.proto
export const lastOccurredAtField = 'Last Updated';
export const numOccurrencesField = 'Event Occurrence';

export const sortFields = [lastOccurredAtField, numOccurrencesField]; // correspond to numOccurrences and lastOccurredAt
export const defaultSortOption: SortOption = {
    field: lastOccurredAtField,
    direction: 'desc', // descending from most recent
};
