import React, { ReactElement, useState } from 'react';
import { format } from 'date-fns';
import {
    Card,
    CardHeader,
    CardTitle,
    CardExpandableContent,
    CardBody,
    DescriptionList,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import dateTimeFormat from 'constants/dateTimeFormat';
import { NetworkFlowInfo } from 'types/alert.proto';

export type NetworkFlowCardProps = {
    networkFlowInfo: NetworkFlowInfo;
    message: string;
    time: string;
};

function NetworkFlowCard({ networkFlowInfo, message, time }: NetworkFlowCardProps): ReactElement {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded(!isExpanded);
    }

    return (
        <div className="pf-u-mb-md" key={message}>
            <Card isExpanded={isExpanded} id={message} isFlat>
                <CardHeader onExpand={onExpand}>
                    <CardTitle>{message}</CardTitle>
                </CardHeader>
                <CardExpandableContent>
                    <CardBody>
                        <DescriptionList>
                            <DescriptionListItem
                                term="Source entity type"
                                desc={networkFlowInfo.source.entityType}
                            />
                            <DescriptionListItem
                                term="Source name"
                                desc={networkFlowInfo.source.name}
                            />
                            {(!!networkFlowInfo?.source?.deploymentType ||
                                !!networkFlowInfo?.source?.deploymentNamespace) && (
                                <>
                                    {!!networkFlowInfo?.source?.deploymentType && (
                                        <DescriptionListItem
                                            term="Source deployment type"
                                            desc={networkFlowInfo.source.deploymentType}
                                        />
                                    )}
                                    {!!networkFlowInfo?.source?.deploymentNamespace && (
                                        <DescriptionListItem
                                            term="Source namespace"
                                            desc={networkFlowInfo.source.deploymentNamespace}
                                        />
                                    )}
                                </>
                            )}
                            <DescriptionListItem
                                term="Destination entity type"
                                desc={networkFlowInfo.destination.entityType}
                            />
                            <DescriptionListItem
                                term="Destination name"
                                desc={networkFlowInfo.destination.name}
                            />
                            {(!!networkFlowInfo?.destination?.deploymentType ||
                                !!networkFlowInfo?.destination?.deploymentNamespace) && (
                                <>
                                    {!!networkFlowInfo?.destination?.deploymentType && (
                                        <DescriptionListItem
                                            term="Destination deployment type"
                                            desc={networkFlowInfo.destination.deploymentType}
                                        />
                                    )}
                                    {!!networkFlowInfo?.destination?.deploymentNamespace && (
                                        <DescriptionListItem
                                            term="Destination namespace"
                                            desc={networkFlowInfo.destination.deploymentNamespace}
                                        />
                                    )}
                                </>
                            )}
                            <DescriptionListItem
                                term="Destination port"
                                desc={networkFlowInfo.destination.port as string}
                            />
                            <DescriptionListItem term="Protocol" desc={networkFlowInfo.protocol} />
                            <DescriptionListItem
                                term="Time"
                                desc={time ? format(time, dateTimeFormat) : 'N/A'}
                            />
                        </DescriptionList>
                    </CardBody>
                </CardExpandableContent>
            </Card>
        </div>
    );
}

export default NetworkFlowCard;
