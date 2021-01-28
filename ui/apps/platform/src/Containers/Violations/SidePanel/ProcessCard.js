import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { getTime, format } from 'date-fns';

import RuntimeViolationCollapsibleCard from 'Containers/Violations/RuntimeViolationCollapsibleCard';
import dateTimeFormat from 'constants/dateTimeFormat';
import ProcessCardContent from './ProcessCardContent';

function ProcessCard({ processes, message }) {
    const [selectedId, selectId] = useState(false);

    function onSelectIdHandler(id) {
        // if the same process id is already selected, remove it
        const result = selectedId && selectedId === id ? null : id;
        selectId(result);
    }

    const timestamps = processes.map((process) => getTime(process.signal.time));
    const firstOccurrenceTimestamp = Math.min(...timestamps);
    const lastOccurrenceTimestamp = Math.max(...timestamps);

    const processesList = processes.map((process) => {
        const { id } = process;
        return (
            <ProcessCardContent
                key={id}
                process={process}
                areAnalystNotesVisible={selectedId === id}
                selectProcessId={onSelectIdHandler}
            />
        );
    });

    return (
        <div className="mb-4" key={message} data-testid="runtime-processes">
            <RuntimeViolationCollapsibleCard title={message}>
                <div>
                    <div className="flex flex-1 bg-primary-100">
                        <div className="w-1/2 p-4 border-r border-base-300 leading-normal">
                            <div className="flex justify-center font-700 italic">
                                First Occurrence:
                            </div>
                            <div className="flex justify-center font-600">
                                {format(firstOccurrenceTimestamp, dateTimeFormat)}
                            </div>
                        </div>
                        <div className="w-1/2 p-4 leading-normal">
                            <div className="flex justify-center font-700 italic">
                                Last Occurrence:
                            </div>
                            <div className="flex justify-center font-600">
                                {format(lastOccurrenceTimestamp, dateTimeFormat)}
                            </div>
                        </div>
                    </div>
                    <div>{processesList}</div>
                </div>
            </RuntimeViolationCollapsibleCard>
        </div>
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
