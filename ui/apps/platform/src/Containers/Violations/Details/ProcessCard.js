import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { getTime, format } from 'date-fns';
import {
    Card,
    CardHeader,
    CardBody,
    CardExpandableContent,
    DescriptionList,
} from '@patternfly/react-core';

import DescriptionListItem from 'Components/DescriptionListItem';
import dateTimeFormat from 'constants/dateTimeFormat';
import ProcessCardContent from './ProcessCardContent';

function ProcessCard({ processes, message }) {
    const [selectedId, selectId] = useState(false);
    const [isExpanded, setIsExpanded] = useState(true);

    function onExpand() {
        setIsExpanded(!isExpanded);
    }

    function onSelectIdHandler(id) {
        // if the same process id is already selected, remove it
        const result = selectedId && selectedId === id ? null : id;
        selectId(result);
    }

    const timestamps = processes.map((process) => getTime(process.signal.time));
    const firstOccurrenceTimestamp = Math.min(...timestamps);
    const lastOccurrenceTimestamp = Math.max(...timestamps);

    return (
        <Card isFlat isExpanded={isExpanded}>
            <CardHeader onExpand={onExpand}>{message}</CardHeader>
            <CardExpandableContent>
                <CardBody>
                    <DescriptionList
                        columnModifier={{
                            default: '2Col',
                        }}
                        className="pf-u-my-md"
                    >
                        <DescriptionListItem
                            term="First occurrence"
                            desc={format(firstOccurrenceTimestamp, dateTimeFormat)}
                        />
                        <DescriptionListItem
                            term="Last occurrence"
                            desc={format(lastOccurrenceTimestamp, dateTimeFormat)}
                        />
                    </DescriptionList>
                    {processes.map((process) => (
                        <>
                            <ProcessCardContent
                                key={process.id}
                                process={process}
                                selectProcessId={onSelectIdHandler}
                            />
                        </>
                    ))}
                </CardBody>
            </CardExpandableContent>
        </Card>
    );
}

ProcessCard.propTypes = {
    message: PropTypes.string.isRequired,
    processes: PropTypes.arrayOf(
        PropTypes.shape({
            id: PropTypes.string.isRequired,
        })
    ).isRequired,
};

export default ProcessCard;
