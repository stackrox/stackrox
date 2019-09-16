import React from 'react';
import PropTypes from 'prop-types';

import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

function KeyValue({ key, value }) {
    return (
        <div>
            <span className="font-700">{key}</span> {value}
        </div>
    );
}

KeyValue.propTypes = {
    key: PropTypes.string.isRequired,
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
                <KeyValue key="Ancestors:" value={lineage.join(', ')} />
            </div>
        );
    }
    return (
        <div className="border-t border-base-300" key={process.id}>
            <div className="flex text-base-600">
                <span className="py-2 px-2 bg-caution-200">{execFilePath}</span>
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2 justify-between">
                <KeyValue key="Container ID:" value={containerId} />
                <KeyValue key="Time:" value={timeFormat} />
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue key="User ID:" value={uid} />
            </div>
            <div className="flex flex-1 text-base-600 px-4 py-2">
                <KeyValue key="Arguments:" value={args} />
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
