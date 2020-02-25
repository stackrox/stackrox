import React from 'react';
import PropTypes from 'prop-types';

import ProcessesCollapsibleCard from 'Containers/Violations/ProcessesCollapsibleCard';

import { getTime, format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';
import ProcessMessage from './ProcessMessage';

function RuntimeMessages({ processViolation }) {
    if (processViolation === null) {
        return null;
    }

    const { message, processes } = processViolation;
    if (!processes.length) {
        return null;
    }

    const timestamps = processes.map(process => getTime(process.signal.time));
    const firstOccurrenceTimestamp = Math.min(...timestamps);
    const lastOccurrenceTimestamp = Math.max(...timestamps);

    const processesList = processes.map(process => <ProcessMessage process={process} />);

    return (
        <div className="mb-4" key={message} data-testid="runtime-processes">
            <ProcessesCollapsibleCard title={message}>
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
            </ProcessesCollapsibleCard>
        </div>
    );
}

RuntimeMessages.propTypes = {
    processViolation: PropTypes.shape({
        message: PropTypes.string.isRequired,
        processes: PropTypes.array.isRequired
    })
};

RuntimeMessages.defaultProps = {
    processViolation: null
};

export default RuntimeMessages;
