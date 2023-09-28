import axios from './instance';

import { Pagination } from './types';

const eventsCountUrl = '/v1/count/administration/events';
const eventsUrl = '/v1/administration/events';

export type AdministrationEvent = {
    id: string;
    type: AdministrationEventType;
    level: AdministrationEventLevel;
    message: string;
    hint: string;
    domain: string;
    resourceType: string;
    resourceId: string;
    numOccurrences: string; // int64
    lastOccurredAt: string; // ISO 8601
    createdAt: string; // ISO 8601
};

export type AdministrationEventType =
    | 'ADMINISTRATION_EVENT_TYPE_UNKNOWN'
    | 'ADMINISTRATION_EVENT_TYPE_GENERIC'
    | 'ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE';

export type AdministrationEventLevel =
    | 'ADMINISTRATION_EVENT_LEVEL_UNKNOWN'
    | 'ADMINISTRATION_EVENT_LEVEL_INFO'
    | 'ADMINISTRATION_EVENT_LEVEL_SUCCESS'
    | 'ADMINISTRATION_EVENT_LEVEL_WARNING'
    | 'ADMINISTRATION_EVENT_LEVEL_ERROR';

export type AdministrationEventsFilter = {
    from: string; // ISO 8601 lower (older) boundary
    until: string; // ISO 8601 upper (newer) boundary
    domain: string;
    resourceType: string;
    type: AdministrationEventType;
    level: AdministrationEventLevel;
};

export type CountAdministrationEventsRequest = {
    filter: AdministrationEventsFilter;
};

// The total number of notifications after filtering and deduplication.
export type CountAdministrationEventsResponse = {
    count: string; // int64
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

// TODO CountAdministrationEventsRequest
export function countAdministrationEvents(): Promise<string> {
    return axios
        .get<CountAdministrationEventsResponse>(eventsCountUrl)
        .then((response) => response.data.count);
}

export function getAdministrationEvent(id: string): Promise<AdministrationEvent> {
    return axios
        .get<GetAdministrationEventResponse>(`${eventsUrl}/${id}`)
        .then((response) => response.data.event);
}

// TODO ListAdministrationEventsRequest
export function listAdministrationEvents(): Promise<AdministrationEvent[]> {
    return axios
        .get<ListAdministrationEventsResponse>(eventsUrl)
        .then((response) => response.data.events);
}
