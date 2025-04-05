import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import { lockUnlockProcesses } from 'services/ProcessesService';

import ProcessBaselineElementList from './ProcessBaselineElementList';

const lockTooltipText =
    'Locking a container specification process baseline will trigger a violation if an abnormal process is found';
const unlockTooltipText =
    'Unlocking a container specification process baseline will NOT trigger a violation if an abnormal process is found';
const buttonClassName = 'p-1 border rounded';
const disabledButtonClassName =
    'text-primary-400 border-primary-400 bg-base-300 pointer-events-none';
const lockedClassName = `${disabledButtonClassName} rounded-r-none border-base-500`;
const unlockedClassName = `${disabledButtonClassName} rounded-l-none border-base-500`;

const ProcessBaselineList = ({ process, processEpoch, setProcessEpoch }) => {
    const isLocked = !!process.userLockedTimestamp;
    const { key, elements, containerName } = process;
    function toggleCurrentProcessLock() {
        const desiredLocked = !isLocked;
        const query = {
            keys: [{ ...key }],
            locked: desiredLocked,
        };
        lockUnlockProcesses(query).then(() => {
            // This is so that the parent component knows that one of the child components
            // modified the state server side and knows to re-render. Updating the processEpoch
            // value is just a way of causing the parent to reload the data from the server
            // and rerender all of the children.
            setProcessEpoch(processEpoch + 1);
        });
    }

    const sortedElements = elements.sort((a, b) => {
        if (!a.element || !a.element.processName) {
            return -1;
        }
        if (!b.element || !b.element.processName) {
            return 1;
        }
        return a.element.processName.localeCompare(b.element.processName);
    });
    return (
        <li
            key={containerName}
            className="bg-base-100 text-base-600 rounded border border-base-400"
        >
            <div className="text-base-600 font-700 flex justify-between items-center border-b border-base-300 p-3">
                <span>{key.containerName}</span>
                <Tooltip content={isLocked ? unlockTooltipText : lockTooltipText}>
                    <div>
                        <button
                            className={`${buttonClassName} ${
                                isLocked
                                    ? lockedClassName
                                    : 'border-r-0 border-base-500 rounded-r-none hover:bg-base-300'
                            }`}
                            type="button"
                            onClick={toggleCurrentProcessLock}
                            aria-label="Lock baseline"
                        >
                            <Icon.Lock className="h-4 w-4" />
                        </button>
                        <button
                            className={`${buttonClassName} ${
                                !isLocked
                                    ? unlockedClassName
                                    : 'border-l-0 border-base-500 rounded-l-none hover:bg-base-300'
                            }`}
                            type="button"
                            onClick={toggleCurrentProcessLock}
                            aria-label="Unlock baseline"
                        >
                            <Icon.Unlock className="h-4 w-4" />
                        </button>
                    </div>
                </Tooltip>
            </div>
            <ProcessBaselineElementList
                baselineKey={key}
                elements={sortedElements}
                processEpoch={processEpoch}
                setProcessEpoch={setProcessEpoch}
            />
        </li>
    );
};

ProcessBaselineList.propTypes = {
    process: PropTypes.shape({
        key: PropTypes.shape({
            containerName: PropTypes.string,
        }).isRequired,
        elements: PropTypes.arrayOf(PropTypes.object).isRequired,
        containerName: PropTypes.func,
        userLockedTimestamp: PropTypes.string,
    }).isRequired,
    processEpoch: PropTypes.number.isRequired,
    setProcessEpoch: PropTypes.func.isRequired,
};

export default ProcessBaselineList;
