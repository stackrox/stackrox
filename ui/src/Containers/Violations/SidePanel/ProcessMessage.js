import React from 'react';
import PropTypes from 'prop-types';
import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';

function KeyValue({ label, value }) {
    return (
        <div>
            <span className="font-700">{label}</span> {value}
        </div>
    );
}

KeyValue.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.string.isRequired
};

function ProcessMessage({ process }) {
    const { time, args, execFilePath, containerId, lineage, uid } = process.signal;
    const processTime = new Date(time);
    const timeFormat = format(processTime, dateTimeFormat);
    let ancestors = null;
    if (Array.isArray(lineage) && lineage.length) {
        ancestors = (
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue label="Ancestors:" value={lineage.join(', ')} />
            </div>
        );
    }
    return (
        <div className="border-t border-base-300" label={process.id}>
            <div className="flex text-base-600">
                <span className="py-2 px-2 bg-caution-200">{execFilePath}</span>
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2 justify-between">
                <KeyValue label="Container ID:" value={containerId} />
                <KeyValue label="Time:" value={timeFormat} />
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue label="User ID:" value={uid} />
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue label="Arguments:" value={args} />
            </div>
            {ancestors}
        </div>
    );
}

ProcessMessage.propTypes = {
    process: PropTypes.shape({
        id: PropTypes.string.isRequired,
        signal: PropTypes.shape({
            time: PropTypes.string.isRequired,
            args: PropTypes.string.isRequired,
            execFilePath: PropTypes.string.isRequired,
            containerId: PropTypes.string.isRequired,
            lineage: PropTypes.arrayOf(PropTypes.string).isRequired,
            uid: PropTypes.string.isRequired
        })
    }).isRequired
};

export default ProcessMessage;
