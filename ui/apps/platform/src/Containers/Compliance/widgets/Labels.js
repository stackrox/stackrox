import React from 'react';
import PropTypes from 'prop-types';
import { Tooltip } from '@patternfly/react-core';

const truncate = (key) => {
    const index = key.indexOf('/');
    return index > 0 ? key.substr(index + 1) : key;
};

const Labels = ({ labels }) => (
    <ul className={` ${labels.length > 4 ? `columns-2` : ``} p-3 w-full leading-normal`}>
        {labels.map((label) => (
            <li
                key={label.key}
                className="border-b border-base-300 p-2 truncate"
                style={{
                    columnBreakInside: 'avoid',
                    pageBreakInside: 'avoid',
                }}
            >
                <Tooltip content={`${label.key} : ${label.value || '""'}`}>
                    <span className="text-base word-break truncate">
                        <span className="font-700 pr-1">{`${truncate(label.key)}:`}</span>
                        <span>{label.value || '""'}</span>
                    </span>
                </Tooltip>
            </li>
        ))}
    </ul>
);

Labels.propTypes = {
    labels: PropTypes.arrayOf(PropTypes.object).isRequired,
};

export default Labels;
