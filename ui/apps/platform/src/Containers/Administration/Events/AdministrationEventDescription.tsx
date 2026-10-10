import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Alert,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
} from '@patternfly/react-core';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import type { AdministrationEvent } from 'services/AdministrationEventsService';

import { getLevelText, getLevelVariant, getTypeText } from './AdministrationEvent';
import AdministrationEventHintMessage from './AdministrationEventHintMessage';
import RelatedDeployments from './RelatedDeployments';

export type AdministrationEventDescriptionProps = {
    event: AdministrationEvent;
};

function AdministrationEventDescription({
    event,
}: AdministrationEventDescriptionProps): ReactElement {
    const { createdAt, id, lastOccurredAt, level, numOccurrences, resource, type } = event;
    const { id: resourceID, name: resourceName, type: resourceType } = resource;

    return (
        <Flex direction={{ default: 'column' }}>
            <Alert
                component="p"
                isInline
                title={getLevelText(level)}
                variant={getLevelVariant(level)}
            >
                <AdministrationEventHintMessage event={event} />
            </Alert>
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
                        <DescriptionListDescription>
                            {resourceType === 'Image' ? (
                                <Link
                                    to={`${vulnerabilitiesWorkloadCvesPath}/images/${resourceID}`}
                                >
                                    {resourceID}
                                </Link>
                            ) : (
                                resourceID
                            )}
                        </DescriptionListDescription>
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
                    <DescriptionListTerm>Created</DescriptionListTerm>
                    <DescriptionListDescription>{createdAt}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Last occurred</DescriptionListTerm>
                    <DescriptionListDescription>{lastOccurredAt}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Count</DescriptionListTerm>
                    <DescriptionListDescription>{numOccurrences}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            {resourceType === 'Image' && resourceID && <RelatedDeployments imageId={resourceID} />}
        </Flex>
    );
}

export default AdministrationEventDescription;
