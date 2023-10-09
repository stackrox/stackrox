import React, { ReactElement, useEffect, useState } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Flex,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { AdministrationEvent, getAdministrationEvent } from 'services/AdministrationEventsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { administrationEventsBasePath } from 'routePaths';

import AdministrationEventDescription from './AdministrationEventDescription';

export type AdministrationEventPageProps = {
    id: string;
};

function AdministrationEventPage({ id }: AdministrationEventPageProps): ReactElement {
    const [isLoading, setIsLoading] = useState(false);
    const [event, setEvent] = useState<AdministrationEvent | null>(null);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);
        getAdministrationEvent(id)
            .then((eventArg) => {
                setEvent(eventArg);
                setErrorMessage('');
            })
            .catch((error) => {
                setEvent(null);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [id]);

    const h1 = event ? event.domain : 'Administration event';

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageTitle title={`Administration events - ${h1}`} />
            <PageSection component="div" variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Breadcrumb>
                        <BreadcrumbItemLink to={administrationEventsBasePath}>
                            Administration events
                        </BreadcrumbItemLink>
                        <BreadcrumbItem>{h1}</BreadcrumbItem>
                    </Breadcrumb>
                    <Title headingLevel="h1">{h1}</Title>
                </Flex>
            </PageSection>
            <PageSection component="div" variant="light">
                {isLoading ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : errorMessage ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch administration event"
                        component="div"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : event ? (
                    <AdministrationEventDescription event={event} />
                ) : null}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default AdministrationEventPage;
