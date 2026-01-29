import { useState } from 'react';
import type { ComponentType, ReactElement } from 'react';
import { getTime } from 'date-fns';
import {
    Card,
    CardBody,
    CardExpandableContent,
    CardHeader,
    DescriptionList,
    Flex,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import { getDateTime } from 'utils/dateUtils';

type TimestampedEventCardProps<T> = {
    message: string;
    events: T[];
    getTimestamp: (event: T) => string | null;
    ContentComponent: ComponentType<{ event: T }>;
    getEventKey: (event: T) => string;
};

function TimestampedEventCard<T>({
    message,
    events,
    getTimestamp,
    ContentComponent,
    getEventKey,
}: TimestampedEventCardProps<T>): ReactElement {
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded((prev) => !prev);
    }

    const timestamps = events
        .map((event) => getTimestamp(event))
        .filter((timestamp): timestamp is string => timestamp !== null)
        .map((timestamp) => getTime(timestamp));
    const firstOccurrenceTimestamp = Math.min(...timestamps);
    const lastOccurrenceTimestamp = Math.max(...timestamps);

    return (
        <Card isFlat isExpanded={isExpanded}>
            <CardHeader
                onExpand={onExpand}
                toggleButtonProps={{ 'aria-expanded': isExpanded, 'aria-label': 'Details' }}
            >
                {message}
            </CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <DescriptionList
                        columnModifier={{
                            default: '2Col',
                        }}
                        className="pf-v5-u-my-md"
                    >
                        <DescriptionListItem
                            term="First occurrence"
                            desc={getDateTime(firstOccurrenceTimestamp)}
                        />
                        <DescriptionListItem
                            term="Last occurrence"
                            desc={getDateTime(lastOccurrenceTimestamp)}
                        />
                    </DescriptionList>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        {events.map((event) => (
                            <ContentComponent key={getEventKey(event)} event={event} />
                        ))}
                    </Flex>
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
}

export default TimestampedEventCard;
