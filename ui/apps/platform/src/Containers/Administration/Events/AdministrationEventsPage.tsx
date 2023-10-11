import React, { ReactElement, useCallback, useEffect, useState } from 'react';
import { Alert, Bullseye, PageSection, Spinner, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useInterval from 'hooks/useInterval';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import {
    AdministrationEvent,
    countAdministrationEvents,
    defaultSortOption,
    getListAdministrationEventsArg,
    listAdministrationEvents,
    sortFields,
} from 'services/AdministrationEventsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import AdministrationEventsTable from './AdministrationEventsTable';
import AdministrationEventsToolbar from './AdministrationEventsToolbar';

function AdministrationEventsPage(): ReactElement {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { getSortParams, sortOption } = useURLSort({ defaultSortOption, sortFields });

    const [count, setCount] = useState(0);
    const [countAvailable, setCountAvailable] = useState(0);
    const [errorMessage, setErrorMessage] = useState('');
    const [events, setEvents] = useState<AdministrationEvent[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [lastUpdatedEventIds, setLastUpdatedEventIds] = useState<Set<string>>(new Set());
    const [lastUpdatedTime, setLastUpdatedTime] = useState('');

    /*
     * Request count and events at initial page load or after user action:
     * 1. change pagination, searchFilter, sortOptions:
     *    * because useCallback hook dependencies change,
     *    * it returns a new updateEvents callback function,
     *    * which is the useEffect hook dependency.
     * 2. click events available button which has `onClick={updateEvents}` prop.
     */
    const updateEvents = useCallback(() => {
        setIsLoading(true);

        const listArg = getListAdministrationEventsArg({ page, perPage, searchFilter, sortOption });
        const { filter } = listArg;

        Promise.all([countAdministrationEvents(filter), listAdministrationEvents(listArg)])
            .then(([countArg, eventsArg]) => {
                setCount(countArg);
                setCountAvailable(0);
                setErrorMessage('');
                setEvents(eventsArg);
                setLastUpdatedEventIds(new Set(eventsArg.map(({ id }) => id)));
                setLastUpdatedTime(new Date().toISOString());
            })
            .catch((error) => {
                setCount(0);
                setErrorMessage(getAxiosErrorMessage(error));
                setEvents([]);
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [page, perPage, searchFilter, setIsLoading, sortOption]);

    useEffect(() => {
        updateEvents();
    }, [updateEvents]);

    /*
     * Request events every minute to compute count of events available.
     */
    useInterval(() => {
        const arg = getListAdministrationEventsArg({ page, perPage, searchFilter, sortOption });
        listAdministrationEvents(arg)
            .then((eventsArg) => {
                setCountAvailable(
                    eventsArg.reduce(
                        (countAvailableAccumulator, { id }) =>
                            lastUpdatedEventIds.has(id)
                                ? countAvailableAccumulator
                                : countAvailableAccumulator + 1,
                        0
                    )
                );
            })
            .catch(() => {});
    }, 60000); // 60 seconds corresponds to backend reprocessing events.

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageTitle title="Administration Events" />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Administration Events</Title>
                <Text>
                    Troubleshoot platform issues by reviewing event logs. Events are purged after 4
                    days by default.
                </Text>
            </PageSection>
            <PageSection component="div">
                {isLoading && !lastUpdatedTime ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : errorMessage ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch administration events"
                        component="div"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : (
                    <>
                        <AdministrationEventsToolbar
                            count={count}
                            countAvailable={countAvailable}
                            isDisabled={isLoading}
                            lastUpdatedTime={lastUpdatedTime}
                            page={page}
                            perPage={perPage}
                            setPage={setPage}
                            setPerPage={setPerPage}
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                            updateEvents={updateEvents}
                        />
                        <AdministrationEventsTable
                            events={events}
                            getSortParams={getSortParams}
                            searchFilter={searchFilter}
                        />
                    </>
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default AdministrationEventsPage;
