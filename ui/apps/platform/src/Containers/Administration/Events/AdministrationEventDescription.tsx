import React, { ReactElement } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
} from '@patternfly/react-core';

import { AdministrationEvent } from 'services/AdministrationEventsService';

import { getTypeText } from './AdministrationEvent';

export type AdministrationEventDescriptionProps = {
    event: AdministrationEvent;
};

function AdministrationEventDescription({
    event,
}: AdministrationEventDescriptionProps): ReactElement {
    const { createdAt, id, lastOccurredAt, numOccurrences, resource, type } = event;
    const { id: resourceID, name: resourceName, type: resourceType } = resource;

    // TODO render hint and message when page design is ready.
    // TODO factor out if same presentation in page and table.
    // TODO render optional resourceName when it has been added to response.
    return (
        <Flex direction={{ default: 'column' }}>
            <DescriptionList isCompact isHorizontal>
                <DescriptionListGroup>
                    <DescriptionListTerm>Resource type</DescriptionListTerm>
                    <DescriptionListDescription>{resourceType}</DescriptionListDescription>
                </DescriptionListGroup>
                {resourceName && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Resource name</DescriptionListTerm>
                        <DescriptionListDescription>{resourceName}</DescriptionListDescription>
                    </DescriptionListGroup>
                )}
                {resourceID && (
                    <DescriptionListGroup>
                        <DescriptionListTerm>Resource ID</DescriptionListTerm>
                        <DescriptionListDescription>{resourceID}</DescriptionListDescription>
                    </DescriptionListGroup>
                )}
                <DescriptionListGroup>
                    <DescriptionListTerm>Event type</DescriptionListTerm>
                    <DescriptionListDescription>{getTypeText(type)}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Event ID</DescriptionListTerm>
                    <DescriptionListDescription>{id}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Created at</DescriptionListTerm>
                    <DescriptionListDescription>{createdAt}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Last occurred at</DescriptionListTerm>
                    <DescriptionListDescription>{lastOccurredAt}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Count</DescriptionListTerm>
                    <DescriptionListDescription>{numOccurrences}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
        </Flex>
    );
}

export default AdministrationEventDescription;
