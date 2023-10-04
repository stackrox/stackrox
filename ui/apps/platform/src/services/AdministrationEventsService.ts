import qs from 'qs';

import { SearchFilter } from 'types/search';
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

export type AdministrationEventType =
    | 'ADMINISTRATION_EVENT_TYPE_UNKNOWN'
    | 'ADMINISTRATION_EVENT_TYPE_GENERIC'
    | 'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE';

const types: AdministrationEventType[] = [
    'ADMINISTRATION_EVENT_TYPE_UNKNOWN',
    'ADMINISTRATION_EVENT_TYPE_GENERIC',
    'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE',
]; // for isType function

export type AdministrationEventLevel =
    | 'ADMINISTRATION_EVENT_LEVEL_UNKNOWN'
    | 'ADMINISTRATION_EVENT_LEVEL_INFO'
    | 'ADMINISTRATION_EVENT_LEVEL_SUCCESS'
    | 'ADMINISTRATION_EVENT_LEVEL_WARNING'
    | 'ADMINISTRATION_EVENT_LEVEL_ERROR';

const levels: AdministrationEventLevel[] = [
    'ADMINISTRATION_EVENT_LEVEL_UNKNOWN',
    'ADMINISTRATION_EVENT_LEVEL_INFO',
    'ADMINISTRATION_EVENT_LEVEL_SUCCESS',
    'ADMINISTRATION_EVENT_LEVEL_WARNING',
    'ADMINISTRATION_EVENT_LEVEL_ERROR',
]; // for isLevel function

export type AdministrationEventsFilter = {
    from?: string; // ISO 8601 lower (older) boundary
    until?: string; // ISO 8601 upper (newer) boundary
    domain?: string[];
    resourceType?: string[];
    type?: AdministrationEventType[];
    level?: AdministrationEventLevel[];
};

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
    pagination?: Pagination;
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

// Given searchFilter from useURLSearch hook, return validated filter argument.

export function getAdministrationEventsFilter(
    searchFilter: SearchFilter
): AdministrationEventsFilter {
    // For consistency with useURLSort hook, especially in case the table columns become sortable,
    // useURLSearch hook also uses search strings.
    // See proto/storage/administration_event.proto
    return {
        domain: getValue(searchFilter['Event Domain']),
        level: getLevel(searchFilter['Event Level']),
        resourceType: getValue(searchFilter['Resource Type']),
        type: getType(searchFilter['Event Type']),
    };
}

type SearchFilterValue = string | string[] | undefined;

function getLevel(arg: SearchFilterValue): AdministrationEventLevel[] | undefined {
    if (typeof arg === 'string' && isLevel(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter((item) => isLevel(item)) as AdministrationEventLevel[];
    }

    return undefined;
}

function isLevel(arg: string): arg is AdministrationEventLevel {
    return levels.includes(arg as AdministrationEventLevel);
}

function getType(arg: SearchFilterValue): AdministrationEventType[] | undefined {
    if (typeof arg === 'string' && isType(arg)) {
        return [arg];
    }

    if (Array.isArray(arg)) {
        return arg.filter((item) => isType(item)) as AdministrationEventType[];
    }

    return undefined;
}

function isType(arg: string): arg is AdministrationEventType {
    return types.includes(arg as AdministrationEventType);
}

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

export const lastOccurredAtField = 'Last Updated';
export const numOccurrencesField = 'Event Occurrence';

export const sortFields = [lastOccurredAtField, numOccurrencesField]; // correspond to numOccurrences and lastOccurredAt
export const defaultSortOption: SortOption = {
    field: lastOccurredAtField,
    direction: 'desc', // descending from most recent
};
