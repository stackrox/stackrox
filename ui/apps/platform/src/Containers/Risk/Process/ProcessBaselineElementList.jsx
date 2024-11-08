import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import { addDeleteProcesses } from 'services/ProcessesService';

const ProcessBaselineElementList = ({ baselineKey, elements, processEpoch, setProcessEpoch }) => {
    if (!elements || !elements.length) {
        return <span className="p-3 block"> No elements in this baseline </span>;
    }

    function deleteCurrentProcess(element) {
        return () => {
            const query = {
                keys: [{ ...baselineKey }],
                removeElements: [element],
            };
            addDeleteProcesses(query).then(() => {
                setProcessEpoch(processEpoch + 1);
            });
        };
    }

    return (
        <ul className="pl-3 pr-3">
            {elements.map(({ element }) => (
                <li
                    key={element.processName}
                    className="py-3 pb-2 leading-normal border-b border-base-300 flex justify-between items-center"
                >
                    <span>{element.processName}</span>
                    <Tooltip content="Remove process from baseline">
                        <button
                            className="flex p-1 rounded border content-center hover:bg-base-300"
                            type="button"
                            onClick={deleteCurrentProcess(element)}
                            aria-label="Remove process from baseline"
                        >
                            <Icon.Minus className="h-4 w-4" />
                        </button>
                    </Tooltip>
                </li>
            ))}
        </ul>
    );
};

ProcessBaselineElementList.propTypes = {
    elements: PropTypes.arrayOf(
        PropTypes.shape({
            processName: PropTypes.string,
        })
    ),
    baselineKey: PropTypes.shape({}),
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
};

ProcessBaselineElementList.defaultProps = {
    elements: [],
    baselineKey: {},
};

export default ProcessBaselineElementList;
