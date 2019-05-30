import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { Tooltip } from 'react-tippy';
import { actions as processesActions } from 'reducers/processes';

const lockTooltipText =
    'Locking a container specification whitelist will trigger a violation if an abnormal process is found';
const unlockTooltipText =
    'Unlocking a container specification whitelist will NOT trigger a violation if an abnormal process is found';
const buttonClassName = 'p-1 border rounded';
const disabledButtonClassName =
    'text-primary-400 border-primary-400 bg-base-300 pointer-events-none';
const lockedClassName = `${disabledButtonClassName} rounded-r-none border-base-500`;
const unlockedClassName = `${disabledButtonClassName} rounded-l-none border-base-500`;

const WhitelistElementsList = ({ whitelistKey, elements, deleteProcess }) => {
    if (!elements || !elements.length) {
        return <span className="p-3 block"> No elements in this whitelist </span>;
    }

    const deleteCurrentProcess = element => () => {
        if (deleteProcess) {
            const query = {
                keys: [{ ...whitelistKey }],
                removeElements: [element]
            };
            deleteProcess(query);
        }
    };

    return (
        <ul className="list-reset pl-3 pr-3">
            {elements.map(({ element }) => (
                <li
                    key={element.processName}
                    className="py-3 pb-2 leading-normal tracking-normal border-b border-base-300 flex justify-between items-center"
                >
                    <span>{element.processName}</span>
                    <Tooltip
                        useContext
                        position="top"
                        trigger="mouseenter"
                        animation="none"
                        duration={0}
                        arrow
                        html={<span className="text-sm">Remove process from whitelist</span>}
                        unmountHTMLWhenHide
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

WhitelistElementsList.propTypes = {
    elements: PropTypes.arrayOf(
        PropTypes.shape({
            processName: PropTypes.string
        })
    ),
    whitelistKey: PropTypes.shape({}),
    deleteProcess: PropTypes.func
};

WhitelistElementsList.defaultProps = {
    elements: [],
    whitelistKey: {},
    deleteProcess: null
};

const ProcessWhitelist = ({ process, deleteProcess, toggleProcessLock }) => {
    const { key, elements, containerName, userLockedTimestamp } = process;
    const toggleCurrentProcessLock = lockState => () => {
        const query = {
            keys: [{ ...key }],
            locked: lockState
        };
        toggleProcessLock(query);
    };

    const isLocked = userLockedTimestamp;

    return (
        <li
            key={containerName}
            className="bg-base-100 text-base-600 rounded border border-base-400 mb-3"
        >
            <div className="text-base-600 font-700 text-lg flex justify-between items-center border-b border-base-300 p-3">
                <span>{key.containerName}</span>
                <Tooltip
                    useContext
                    position="top"
                    trigger="mouseenter"
                    animation="none"
                    duration={0}
                    arrow
                    html={
                        <span className="text-sm">
                            {isLocked ? unlockTooltipText : lockTooltipText}
                        </span>
                    }
                    unmountHTMLWhenHide
                >
                    <div>
                        <button
                            className={`${buttonClassName} ${
                                isLocked
                                    ? lockedClassName
                                    : 'border-r-0 border-base-500 rounded-r-none hover:bg-base-300'
                            }`}
                            type="button"
                            onClick={toggleCurrentProcessLock(true)}
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
                            onClick={toggleCurrentProcessLock(false)}
                        >
                            <Icon.Unlock className="h-4 w-4" />
                        </button>
                    </div>
                </Tooltip>
            </div>
            <WhitelistElementsList
                whitelistKey={key}
                elements={elements}
                deleteProcess={deleteProcess}
            />
        </li>
    );
};

ProcessWhitelist.propTypes = {
    process: PropTypes.shape({}).isRequired,
    deleteProcess: PropTypes.func.isRequired,
    toggleProcessLock: PropTypes.func.isRequired
};

const mapDispatchToProps = {
    deleteProcess: processesActions.addDeleteProcesses,
    toggleProcessLock: processesActions.lockUnlockProcesses
};

export default connect(
    null,
    mapDispatchToProps
)(ProcessWhitelist);
