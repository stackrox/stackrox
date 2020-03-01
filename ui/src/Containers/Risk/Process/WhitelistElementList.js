import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';
import { addDeleteProcesses } from 'services/ProcessesService';

const ProcessWhitelistElementsList = ({
    whitelistKey,
    elements,
    processEpoch,
    setProcessEpoch
}) => {
    if (!elements || !elements.length) {
        return <span className="p-3 block"> No elements in this whitelist </span>;
    }

    function deleteCurrentProcess(element) {
        return () => {
            const query = {
                keys: [{ ...whitelistKey }],
                removeElements: [element]
            };
            addDeleteProcesses(query).then(() => {
                setProcessEpoch(processEpoch + 1);
            });
        };
    }

    return (
        <ul className="list-reset pl-3 pr-3">
            {elements.map(({ element }) => (
                <li
                    key={element.processName}
                    className="py-3 pb-2 leading-normal tracking-normal border-b border-base-300 flex justify-between items-center"
                >
                    <span>{element.processName}</span>
                    <Tooltip
                        content={<TooltipOverlay>Remove process from whitelist</TooltipOverlay>}
                    >
                        <button
                            className="flex p-1 rounded border content-center hover:bg-base-300"
                            type="button"
                            onClick={deleteCurrentProcess(element)}
                        >
                            <Icon.Minus className="h-4 w-4" />
                        </button>
                    </Tooltip>
                </li>
            ))}
        </ul>
    );
};

ProcessWhitelistElementsList.propTypes = {
    elements: PropTypes.arrayOf(
        PropTypes.shape({
            processName: PropTypes.string
        })
    ),
    whitelistKey: PropTypes.shape({}),
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired
};

ProcessWhitelistElementsList.defaultProps = {
    elements: [],
    whitelistKey: {}
};

export default ProcessWhitelistElementsList;
