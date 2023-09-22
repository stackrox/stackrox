import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Bullseye, PageSection, Spinner, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import {
    AdministrationEvent,
    countAdministrationEvents,
    listAdministrationEvents,
} from 'services/AdministrationEventsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import AdministrationEventsTable from './AdministrationEventsTable';
import AdministrationEventsToolbar from './AdministrationEventsToolbar';

function AdministrationEventsPage(): ReactElement {
    // TODO query string for table filter and pagination

    const [isLoading, setIsLoading] = useState(false);
    const [events, setEvents] = useState<AdministrationEvent[]>([]);
    const [count, setCount] = useState('0'); // int64
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);
        // TODO Promise.all for consistent count and events?

        listAdministrationEvents()
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

        countAdministrationEvents()
            .then((countArg) => {
                setCount(countArg);
            })
            .catch(() => {
                setCount('0');
            });
    }, [setIsLoading]);

    // TODO empty state with and without filter
    // TODO polling and last updated with conditionally rendered reload button like Network Graph
    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageTitle title="Administration Events" />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Administration Events</Title>
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
                        <AdministrationEventsToolbar count={count} />
                        <AdministrationEventsTable events={events} />
                    </>
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default AdministrationEventsPage;
