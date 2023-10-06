import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Bullseye, PageSection, Spinner, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import {
    AdministrationEvent,
    countAdministrationEvents,
    defaultSortOption,
    getAdministrationEventsFilter,
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

    const [isLoading, setIsLoading] = useState(false);
    const [events, setEvents] = useState<AdministrationEvent[]>([]);
    const [count, setCount] = useState(0);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);

        const filter = getAdministrationEventsFilter(searchFilter);
        const pagination = { limit: perPage, offset: page - 1, sortOption };
        // TODO Promise.all for consistent count and events?

        listAdministrationEvents({ filter, pagination })
            .then((eventsArg) => {
                setEvents(eventsArg);
                setErrorMessage('');
            })
            .catch((error) => {
                setEvents([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });

        countAdministrationEvents(filter)
            .then((countArg) => {
                setCount(countArg);
            })
            .catch(() => {
                setCount(0);
            });
    }, [page, perPage, searchFilter, sortOption, setIsLoading]);

    // TODO polling and last updated with conditionally rendered reload button like Network Graph
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
                            page={page}
                            perPage={perPage}
                            setPage={setPage}
                            setPerPage={setPerPage}
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
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
