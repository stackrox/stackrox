import React, { ReactElement, useEffect, useState } from 'react';
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
    const [errorMessage, setErrorMessage] = useState('');
    const [events, setEvents] = useState<AdministrationEvent[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    const [lastUpdatedEventIds, setLastUpdatedEventIds] = useState<Set<string>>(new Set());
    const [lastUpdatedTime, setLastUpdatedTime] = useState('');
    const [updatedCount, setUpdatedCount] = useState(0);

    const [countAvailable, setCountAvailable] = useState(0);
    const [pollingCount, setPollingCount] = useState(0);

    /*
     * Request count and events at initial page load or after user action:
     * change pagination, searchFilter, sortOptions
     * click events available button
     */
    useEffect(() => {
        if (updatedCount === 0) {
            setIsLoading(true);
        }

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
                if (updatedCount === 0) {
                    setIsLoading(false);
                }
            });
    }, [page, perPage, searchFilter, setIsLoading, sortOption, updatedCount]);

    function updateEvents() {
        setUpdatedCount(updatedCount + 1);
    }

    /*
     * Request events every minute to compute count of events available.
     */
    useEffect(() => {
        if (pollingCount !== 0) {
            const arg = getListAdministrationEventsArg({ page, perPage, searchFilter, sortOption });
            listAdministrationEvents(arg)
                .then((eventsArg) => {
                    let nAvailable = 0;

                    eventsArg.forEach(({ id }) => {
                        if (!lastUpdatedEventIds.has(id)) {
                            nAvailable += 1;
                        }
                    });

                    setCountAvailable(nAvailable);
                })
                .catch(() => {});
        }
    }, [pollingCount]); // eslint-disable-line react-hooks/exhaustive-deps
    // Why disable the rule for the following hook dependencies: lastUpdatedEventIds, page, perPage, searchFilter, sortOption?
    // So polling continues on its schedule instead of immediately making a redundant request.

    useInterval(() => {
        setPollingCount(pollingCount + 1);
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
                {isLoading ? (
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
